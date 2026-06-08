package d2features

import (
	"errors"
	"strings"

	"oss.terrastruct.com/d2/d2ast"
	"oss.terrastruct.com/d2/d2format"
	"oss.terrastruct.com/d2/d2lsp"
	"oss.terrastruct.com/d2/d2parser"
)

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type DocumentHighlight struct {
	Range Range `json:"range"`
	Kind  int   `json:"kind,omitempty"`
}

const documentHighlightKindText = 1

func References(path, text string, line, character int, includeDeclaration bool) ([]Location, error) {
	locations, err := ReferencesInFiles(path, path, map[string]string{path: text}, nil, line, character, includeDeclaration)
	if err != nil {
		return nil, err
	}
	return locations, nil
}

func ReferencesInFiles(defaultURI, path string, fs map[string]string, uriByPath map[string]string, line, character int, includeDeclaration bool) ([]Location, error) {
	text, ok := fs[path]
	if !ok {
		return nil, nil
	}

	locations, err := referenceLocations(defaultURI, path, text, fs, uriByPath, line, character)
	if err != nil {
		return nil, err
	}
	if includeDeclaration || len(locations) == 0 {
		return locations, nil
	}
	return locations[1:], nil
}

func Definition(path, text string, line, character int) (*Location, error) {
	return DefinitionInFiles(path, path, map[string]string{path: text}, nil, line, character)
}

func DefinitionInFiles(defaultURI, path string, fs map[string]string, uriByPath map[string]string, line, character int) (*Location, error) {
	text, ok := fs[path]
	if !ok {
		return nil, nil
	}

	locations, err := referenceLocations(defaultURI, path, text, fs, uriByPath, line, character)
	if err != nil || len(locations) == 0 {
		return nil, err
	}
	return &locations[0], nil
}

func DocumentHighlights(path, text string, line, character int) ([]DocumentHighlight, error) {
	return DocumentHighlightsInFiles(path, path, map[string]string{path: text}, nil, line, character)
}

func DocumentHighlightsInFiles(defaultURI, path string, fs map[string]string, uriByPath map[string]string, line, character int) ([]DocumentHighlight, error) {
	text, ok := fs[path]
	if !ok {
		return []DocumentHighlight{}, nil
	}

	locations, err := referenceLocations(defaultURI, path, text, fs, uriByPath, line, character)
	if err != nil {
		return nil, err
	}

	highlights := make([]DocumentHighlight, 0, len(locations))
	for _, location := range locations {
		if location.URI != defaultURI {
			continue
		}
		highlights = append(highlights, DocumentHighlight{
			Range: location.Range,
			Kind:  documentHighlightKindText,
		})
	}
	return highlights, nil
}

func referenceLocations(defaultURI, path, text string, fs map[string]string, uriByPath map[string]string, line, character int) ([]Location, error) {
	pos := d2ast.Position{Line: line, Column: character, Byte: -1}
	key, ok, err := keyAtPosition(path, text, pos)
	if err != nil || !ok {
		return nil, err
	}

	boardPath, err := d2lsp.GetBoardAtPosition(text, pos)
	if err != nil {
		var pe *d2parser.ParseError
		if !errors.As(err, &pe) {
			return nil, err
		}
	}

	ranges, _, err := d2lsp.GetRefRanges(path, fs, boardPath, key)
	if err != nil {
		return nil, err
	}

	locations := make([]Location, 0, len(ranges))
	for _, r := range ranges {
		locations = append(locations, Location{
			URI:   locationURI(defaultURI, uriByPath, r.Path),
			Range: fromD2Range(r),
		})
	}
	return locations, nil
}

func keyAtPosition(path, text string, pos d2ast.Position) (string, bool, error) {
	ast, err := d2parser.Parse(path, strings.NewReader(text), &d2parser.ParseOptions{
		UTF16Pos: true,
	})
	if err != nil {
		var pe *d2parser.ParseError
		if !errors.As(err, &pe) {
			return "", false, err
		}
	}
	if ast == nil {
		return "", false, nil
	}

	return keyInMapAtPosition(ast, pos)
}

func keyInMapAtPosition(m *d2ast.Map, pos d2ast.Position) (string, bool, error) {
	for _, node := range m.Nodes {
		if node.MapKey == nil {
			continue
		}

		if node.MapKey.Value.Map != nil && containsPosition(node.MapKey.Value.Map.Range, pos) {
			if key, ok, err := keyInMapAtPosition(node.MapKey.Value.Map, pos); err != nil || ok {
				return key, ok, err
			}
		}

		if key, ok := keyInMapKeyAtPosition(node.MapKey, pos); ok {
			return key, true, nil
		}
	}
	return "", false, nil
}

func keyInMapKeyAtPosition(mk *d2ast.Key, pos d2ast.Position) (string, bool) {
	if mk.Key != nil && containsPosition(mk.Key.Range, pos) {
		return d2format.Format(mk.Key), true
	}

	for _, edge := range mk.Edges {
		if edge.Src != nil && containsPosition(edge.Src.Range, pos) {
			return d2format.Format(edge.Src), true
		}
		if edge.Dst != nil && containsPosition(edge.Dst.Range, pos) {
			return d2format.Format(edge.Dst), true
		}
		if containsPosition(edge.Range, pos) {
			return d2format.Format(edge), true
		}
	}

	if mk.EdgeKey != nil && containsPosition(mk.EdgeKey.Range, pos) {
		return symbolName(mk), true
	}

	if containsPosition(mk.Range, pos) {
		return symbolName(mk), true
	}

	return "", false
}

func containsPosition(r d2ast.Range, pos d2ast.Position) bool {
	return !pos.Before(r.Start) && pos.Before(r.End)
}

func locationURI(defaultURI string, uriByPath map[string]string, rangePath string) string {
	if rangePath == "" {
		return defaultURI
	}
	if uri, ok := uriByPath[rangePath]; ok {
		return uri
	}
	return rangePath
}
