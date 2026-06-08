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
	symbolKindFile     = 1
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
		switch {
		case node.Import != nil:
			if symbol, ok := symbolForImport(node.Import); ok {
				symbols = append(symbols, symbol)
			}
		case node.MapKey != nil:
			symbol := symbolForMapKey(node.MapKey)
			if node.MapKey.Value.Map != nil {
				symbol.Kind = symbolKindObject
				symbol.Children = symbolsForMap(node.MapKey.Value.Map)
			}
			if importSymbol, ok := symbolForImport(node.MapKey.Value.Import); ok {
				symbol.Children = append(symbol.Children, importSymbol)
			}
			symbols = append(symbols, symbol)
		}
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

func symbolForImport(imp *d2ast.Import) (DocumentSymbol, bool) {
	if imp == nil || len(imp.Path) == 0 {
		return DocumentSymbol{}, false
	}

	name, ok := importSymbolName(imp)
	if !ok {
		return DocumentSymbol{}, false
	}
	return DocumentSymbol{
		Name:           name,
		Kind:           symbolKindFile,
		Range:          fromD2Range(imp.Range),
		SelectionRange: fromD2Range(importSelectionRange(imp)),
	}, true
}

func importSymbolName(imp *d2ast.Import) (string, bool) {
	parts := make([]string, 0, len(imp.Path))
	for _, part := range imp.Path {
		if part.Unbox() == nil {
			continue
		}
		parts = append(parts, part.Unbox().ScalarString())
	}
	if len(parts) == 0 {
		return "", false
	}

	prefix := "@"
	if imp.Spread {
		prefix = "...@"
	}
	return prefix + strings.Join(parts, "."), true
}

func importSelectionRange(imp *d2ast.Import) d2ast.Range {
	start := imp.Path[0].Unbox().GetRange().Start
	end := imp.Path[len(imp.Path)-1].Unbox().GetRange().End
	return d2ast.Range{
		Path:  imp.Range.Path,
		Start: start,
		End:   end,
	}
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
