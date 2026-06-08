package d2features

import (
	"errors"
	"strings"

	"oss.terrastruct.com/d2/d2ast"
	"oss.terrastruct.com/d2/d2parser"
)

type FoldingRange struct {
	StartLine      int    `json:"startLine"`
	StartCharacter int    `json:"startCharacter,omitempty"`
	EndLine        int    `json:"endLine"`
	EndCharacter   int    `json:"endCharacter,omitempty"`
	Kind           string `json:"kind,omitempty"`
}

const foldingRangeKindRegion = "region"

func FoldingRanges(path, text string) ([]FoldingRange, error) {
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

	var ranges []FoldingRange
	collectMapFolds(ast, true, &ranges)
	return ranges, nil
}

func collectMapFolds(m *d2ast.Map, fileMap bool, ranges *[]FoldingRange) {
	if !fileMap {
		r := fromD2Range(m.Range)
		if r.End.Line > r.Start.Line {
			*ranges = append(*ranges, FoldingRange{
				StartLine:      r.Start.Line,
				StartCharacter: r.Start.Character,
				EndLine:        r.End.Line,
				EndCharacter:   r.End.Character,
				Kind:           foldingRangeKindRegion,
			})
		}
	}

	for _, node := range m.Nodes {
		if node.MapKey != nil && node.MapKey.Value.Map != nil {
			collectMapFolds(node.MapKey.Value.Map, false, ranges)
		}
	}
}
