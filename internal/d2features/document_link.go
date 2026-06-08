package d2features

import (
	"errors"
	"net/url"
	"strings"

	"oss.terrastruct.com/d2/d2ast"
	"oss.terrastruct.com/d2/d2parser"
)

type DocumentLink struct {
	Range  Range  `json:"range"`
	Target string `json:"target,omitempty"`
}

func DocumentLinks(path, text string) ([]DocumentLink, error) {
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

	var links []DocumentLink
	collectDocumentLinks(ast, &links)
	return links, nil
}

func collectDocumentLinks(m *d2ast.Map, links *[]DocumentLink) {
	for _, node := range m.Nodes {
		if node.MapKey == nil {
			continue
		}

		if link, ok := documentLinkForKey(node.MapKey); ok {
			*links = append(*links, link)
		}
		if node.MapKey.Value.Map != nil {
			collectDocumentLinks(node.MapKey.Value.Map, links)
		}
	}
}

func documentLinkForKey(mk *d2ast.Key) (DocumentLink, bool) {
	keyword := strings.ToLower(lastKeyPart(mk))
	if keyword != "link" && keyword != "icon" {
		return DocumentLink{}, false
	}

	scalar, ok := mk.Value.Unbox().(d2ast.Scalar)
	if !ok {
		return DocumentLink{}, false
	}

	target := scalar.ScalarString()
	if !isDocumentLinkTarget(target) {
		return DocumentLink{}, false
	}

	return DocumentLink{
		Range:  fromD2Range(scalar.GetRange()),
		Target: target,
	}, true
}

func isDocumentLinkTarget(target string) bool {
	parsed, err := url.Parse(target)
	if err != nil {
		return false
	}
	switch parsed.Scheme {
	case "http", "https", "file":
		return parsed.Host != "" || parsed.Scheme == "file"
	default:
		return false
	}
}
