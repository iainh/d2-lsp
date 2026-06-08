package d2features

import (
	"errors"
	"fmt"
	"strings"

	"oss.terrastruct.com/d2/d2ast"
	"oss.terrastruct.com/d2/d2parser"
	"oss.terrastruct.com/d2/d2target"
)

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

const markupKindMarkdown = "markdown"

func HoverAt(path, text string, line, character int) (*Hover, error) {
	pos := d2ast.Position{Line: line, Column: character, Byte: -1}
	ast, err := d2parser.Parse(path, strings.NewReader(text), &d2parser.ParseOptions{
		UTF16Pos: true,
	})
	if err != nil {
		var pe *d2parser.ParseError
		if !errors.As(err, &pe) {
			return nil, err
		}
	}
	if ast == nil {
		return nil, nil
	}
	return hoverInMap(ast, pos), nil
}

func hoverInMap(m *d2ast.Map, pos d2ast.Position) *Hover {
	for _, node := range m.Nodes {
		if node.MapKey == nil {
			continue
		}

		if node.MapKey.Value.Map != nil && containsPosition(node.MapKey.Value.Map.Range, pos) {
			if hover := hoverInMap(node.MapKey.Value.Map, pos); hover != nil {
				return hover
			}
		}

		if hover := hoverInMapKey(node.MapKey, pos); hover != nil {
			return hover
		}
	}
	return nil
}

func hoverInMapKey(mk *d2ast.Key, pos d2ast.Position) *Hover {
	if mk.Key != nil {
		for _, part := range mk.Key.Path {
			if containsPosition(part.Unbox().GetRange(), pos) {
				name := part.Unbox().ScalarString()
				if text, ok := keywordHover(name); ok {
					r := fromD2Range(part.Unbox().GetRange())
					return newHover(text, &r)
				}
			}
		}
	}

	if mk.EdgeKey != nil {
		for _, part := range mk.EdgeKey.Path {
			if containsPosition(part.Unbox().GetRange(), pos) {
				name := part.Unbox().ScalarString()
				if text, ok := keywordHover(name); ok {
					r := fromD2Range(part.Unbox().GetRange())
					return newHover(text, &r)
				}
			}
		}
	}

	if mk.Value.Unbox() != nil && containsPosition(mk.Value.Unbox().GetRange(), pos) {
		if text, ok := valueHover(mk, mk.Value.Unbox()); ok {
			r := fromD2Range(mk.Value.Unbox().GetRange())
			return newHover(text, &r)
		}
	}

	return nil
}

func keywordHover(name string) (string, bool) {
	lower := strings.ToLower(name)
	if _, ok := d2ast.StyleKeywords[lower]; ok {
		return fmt.Sprintf("`%s` is a D2 style keyword. Use it under `style` or with dotted style syntax.", lower), true
	}
	if _, ok := d2ast.BoardKeywords[lower]; ok {
		return fmt.Sprintf("`%s` defines a D2 board collection.", lower), true
	}
	if _, ok := d2ast.ReservedKeywordHolders[lower]; ok {
		return fmt.Sprintf("`%s` is a D2 reserved keyword that groups nested settings.", lower), true
	}
	if _, ok := d2ast.CompositeReservedKeywords[lower]; ok {
		return fmt.Sprintf("`%s` is a D2 reserved keyword that can contain nested settings.", lower), true
	}
	if _, ok := d2ast.SimpleReservedKeywords[lower]; ok {
		return fmt.Sprintf("`%s` is a D2 reserved keyword.", lower), true
	}
	return "", false
}

func valueHover(mk *d2ast.Key, value d2ast.Value) (string, bool) {
	scalar, ok := value.(d2ast.Scalar)
	if !ok {
		return "", false
	}

	keyword := lastKeyPart(mk)
	if keyword == "" {
		return "", false
	}

	lowerKeyword := strings.ToLower(keyword)
	valueText := strings.ToLower(scalar.ScalarString())
	switch lowerKeyword {
	case "shape":
		if d2target.IsShape(valueText) {
			return fmt.Sprintf("`%s` is a D2 shape.", valueText), true
		}
		if _, ok := d2target.Arrowheads[valueText]; ok {
			return fmt.Sprintf("`%s` is a D2 arrowhead shape.", valueText), true
		}
	case "near":
		if _, ok := d2ast.NearConstants[valueText]; ok {
			return fmt.Sprintf("`%s` is a D2 `near` constant.", valueText), true
		}
	}

	return "", false
}

func lastKeyPart(mk *d2ast.Key) string {
	switch {
	case mk.EdgeKey != nil && len(mk.EdgeKey.Path) > 0:
		return mk.EdgeKey.Path[len(mk.EdgeKey.Path)-1].Unbox().ScalarString()
	case mk.Key != nil && len(mk.Key.Path) > 0:
		return mk.Key.Path[len(mk.Key.Path)-1].Unbox().ScalarString()
	default:
		return ""
	}
}

func newHover(value string, r *Range) *Hover {
	return &Hover{
		Contents: MarkupContent{
			Kind:  markupKindMarkdown,
			Value: value,
		},
		Range: r,
	}
}
