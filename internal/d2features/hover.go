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

var keywordHoverDescriptions = map[string]string{
	"direction":    "`direction` controls the diagram layout direction. Common values are `up`, `down`, `left`, and `right`.",
	"label":        "`label` sets the text displayed for a shape, connection, or board item.",
	"tooltip":      "`tooltip` adds hover text to rendered diagrams that support tooltips.",
	"link":         "`link` attaches a URL or board path to a shape or connection.",
	"icon":         "`icon` sets an image URL used as the visual icon for a shape.",
	"shape":        "`shape` selects the visual form for an object, such as `rectangle`, `circle`, or `sql_table`.",
	"near":         "`near` positions an object near another object or a supported placement constant.",
	"style":        "`style` groups visual settings such as fill, stroke, opacity, and font size.",
	"classes":      "`classes` declares reusable style classes that can be applied with `class`.",
	"class":        "`class` applies a reusable class declared under `classes`.",
	"vars":         "`vars` declares variables that can be referenced elsewhere in the diagram.",
	"grid-rows":    "`grid-rows` sets the number of rows in a grid diagram.",
	"grid-columns": "`grid-columns` sets the number of columns in a grid diagram.",
}

var styleHoverDescriptions = map[string]string{
	"fill":          "`fill` sets the interior color for a shape.",
	"stroke":        "`stroke` sets the outline color for a shape or connection.",
	"stroke-width":  "`stroke-width` sets outline thickness from 0 to 15.",
	"stroke-dash":   "`stroke-dash` controls dashed outlines or connections.",
	"opacity":       "`opacity` sets transparency from 0.0 to 1.0.",
	"fill-pattern":  "`fill-pattern` sets a texture such as `dots`, `lines`, or `grain`.",
	"font-size":     "`font-size` sets label text size.",
	"font-color":    "`font-color` sets label text color.",
	"border-radius": "`border-radius` rounds rectangle corners.",
	"shadow":        "`shadow` toggles a drop shadow.",
	"3d":            "`3d` toggles supported three-dimensional shape styling.",
}

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
	if text, ok := keywordHoverDescriptions[lower]; ok {
		return text, true
	}
	if text, ok := styleHoverDescriptions[lower]; ok {
		return text, true
	}
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
