package d2features

import (
	"errors"
	"strings"

	"oss.terrastruct.com/d2/d2ast"
	"oss.terrastruct.com/d2/d2format"
	"oss.terrastruct.com/d2/d2parser"
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

const (
	symbolKindProperty = 7
	symbolKindObject   = 19
)

func Symbols(path, text string) ([]DocumentSymbol, error) {
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

	return symbolsForMap(ast), nil
}

func symbolsForMap(m *d2ast.Map) []DocumentSymbol {
	var symbols []DocumentSymbol
	for _, node := range m.Nodes {
		if node.MapKey == nil {
			continue
		}

		symbol := symbolForMapKey(node.MapKey)
		if node.MapKey.Value.Map != nil {
			symbol.Kind = symbolKindObject
			symbol.Children = symbolsForMap(node.MapKey.Value.Map)
		}
		symbols = append(symbols, symbol)
	}
	return symbols
}

func symbolForMapKey(mk *d2ast.Key) DocumentSymbol {
	return DocumentSymbol{
		Name:           symbolName(mk),
		Kind:           symbolKindProperty,
		Range:          fromD2Range(mk.Range),
		SelectionRange: fromD2Range(selectionRange(mk)),
	}
}

func symbolName(mk *d2ast.Key) string {
	nameOnly := &d2ast.Key{
		Range:        mk.Range,
		Ampersand:    mk.Ampersand,
		NotAmpersand: mk.NotAmpersand,
		Key:          mk.Key,
		Edges:        mk.Edges,
		EdgeIndex:    mk.EdgeIndex,
		EdgeKey:      mk.EdgeKey,
	}

	name := d2format.Format(nameOnly)
	if name == "" {
		return "<unknown>"
	}
	return name
}

func selectionRange(mk *d2ast.Key) d2ast.Range {
	switch {
	case mk.Key != nil:
		return mk.Key.Range
	case len(mk.Edges) > 0:
		return mk.Edges[0].Range
	default:
		return mk.Range
	}
}

func fromD2Range(r d2ast.Range) Range {
	return Range{
		Start: fromD2Position(r.Start),
		End:   fromD2Position(r.End),
	}
}

func fromD2Position(pos d2ast.Position) Position {
	return Position{
		Line:      nonnegative(pos.Line),
		Character: nonnegative(pos.Column),
	}
}

func nonnegative(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
