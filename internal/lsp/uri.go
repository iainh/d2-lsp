package lsp

import (
	"net/url"
	"path/filepath"
)

func pathFromURI(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "file" {
		return uri
	}
	if parsed.Host != "" && parsed.Host != "localhost" {
		return filepath.Join(string(filepath.Separator), parsed.Host, parsed.Path)
	}
	return parsed.Path
}

func uriFromPath(path string) string {
	if !filepath.IsAbs(path) {
		return path
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}
