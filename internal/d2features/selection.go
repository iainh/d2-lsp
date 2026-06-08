package d2features

import (
	"errors"
	"sort"
	"strings"

	"oss.terrastruct.com/d2/d2ast"
	"oss.terrastruct.com/d2/d2parser"
)

type SelectionRange struct {
	Range  Range           `json:"range"`
	Parent *SelectionRange `json:"parent,omitempty"`
}

func SelectionRanges(path, text string, positions []Position) ([]*SelectionRange, error) {
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
		return make([]*SelectionRange, len(positions)), nil
	}

	ranges := make([]*SelectionRange, 0, len(positions))
	for _, pos := range positions {
		ranges = append(ranges, selectionRangeAt(ast, d2ast.Position{
			Line:   pos.Line,
			Column: pos.Character,
			Byte:   -1,
		}))
	}
	return ranges, nil
}

func selectionRangeAt(root d2ast.Node, pos d2ast.Position) *SelectionRange {
	var ranges []d2ast.Range
	d2ast.Walk(root, func(node d2ast.Node) bool {
		r := node.GetRange()
		if validSelectionRange(r) && containsPosition(r, pos) {
			ranges = append(ranges, r)
		}
		return true
	})

	ranges = uniqueRanges(ranges)
	sort.SliceStable(ranges, func(i, j int) bool {
		return rangeSmallerThan(ranges[i], ranges[j])
	})

	var parent *SelectionRange
	for i := len(ranges) - 1; i >= 0; i-- {
		parent = &SelectionRange{
			Range:  fromD2Range(ranges[i]),
			Parent: parent,
		}
	}
	return parent
}

func validSelectionRange(r d2ast.Range) bool {
	return r.Start.Line >= 0 &&
		r.Start.Column >= 0 &&
		r.End.Line >= 0 &&
		r.End.Column >= 0 &&
		!sameD2Position(r.Start, r.End)
}

func uniqueRanges(ranges []d2ast.Range) []d2ast.Range {
	unique := make([]d2ast.Range, 0, len(ranges))
	for _, r := range ranges {
		seen := false
		for _, existing := range unique {
			if sameD2Range(r, existing) {
				seen = true
				break
			}
		}
		if !seen {
			unique = append(unique, r)
		}
	}
	return unique
}

func rangeSmallerThan(a, b d2ast.Range) bool {
	if sameD2Range(a, b) {
		return false
	}
	if containsD2Range(b, a) {
		return true
	}
	if containsD2Range(a, b) {
		return false
	}
	if a.Start.Line != b.Start.Line {
		return a.Start.Line > b.Start.Line
	}
	if a.Start.Column != b.Start.Column {
		return a.Start.Column > b.Start.Column
	}
	if a.End.Line != b.End.Line {
		return a.End.Line < b.End.Line
	}
	return a.End.Column < b.End.Column
}

func containsD2Range(outer, inner d2ast.Range) bool {
	return !inner.Start.Before(outer.Start) && !outer.End.Before(inner.End)
}

func sameD2Range(a, b d2ast.Range) bool {
	return sameD2Position(a.Start, b.Start) && sameD2Position(a.End, b.End)
}

func sameD2Position(a, b d2ast.Position) bool {
	return a.Line == b.Line && a.Column == b.Column
}
