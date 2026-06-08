package d2features

import (
	"errors"
	"path/filepath"
	"strings"

	"oss.terrastruct.com/d2/d2ast"
	"oss.terrastruct.com/d2/d2parser"
)

const inlayHintKindParameter = 2

type InlayHint struct {
	Position Position `json:"position"`
	Label    string   `json:"label"`
	Kind     int      `json:"kind,omitempty"`
}

func InlayHints(path, text string) ([]InlayHint, error) {
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
		return []InlayHint{}, nil
	}

	var hints []InlayHint
	collectInlayHintsForMap(path, ast, &hints)
	return hints, nil
}

func collectInlayHintsForMap(path string, m *d2ast.Map, hints *[]InlayHint) {
	if m == nil {
		return
	}
	for _, node := range m.Nodes {
		switch {
		case node.Import != nil:
			if hint, ok := inlayHintForImport(path, node.Import); ok {
				*hints = append(*hints, hint)
			}
		case node.MapKey != nil:
			collectInlayHintsForMapKey(path, node.MapKey, hints)
		}
	}
}

func collectInlayHintsForMapKey(path string, mk *d2ast.Key, hints *[]InlayHint) {
	if mk == nil {
		return
	}
	if mk.Value.Import != nil {
		if hint, ok := inlayHintForImport(path, mk.Value.Import); ok {
			*hints = append(*hints, hint)
		}
	}
	collectInlayHintsForMap(path, mk.Value.Map, hints)
}

func inlayHintForImport(path string, imp *d2ast.Import) (InlayHint, bool) {
	resolved, ok := resolvedImportPath(path, imp)
	if !ok {
		return InlayHint{}, false
	}
	return InlayHint{
		Position: fromD2Position(imp.Range.End),
		Label:    " => " + resolved,
		Kind:     inlayHintKindParameter,
	}, true
}

func resolvedImportPath(path string, imp *d2ast.Import) (string, bool) {
	if imp == nil || len(imp.Path) == 0 {
		return "", false
	}

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

	importPath := filepath.Join(parts...)
	if filepath.Ext(importPath) == "" {
		importPath += ".d2"
	}
	return filepath.Clean(filepath.Join(filepath.Dir(path), importPath)), true
}
