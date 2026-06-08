package lsp

import "testing"

func TestPathFromURIConvertsFileURI(t *testing.T) {
	got := pathFromURI("file:///tmp/diagram.d2")
	if got != "/tmp/diagram.d2" {
		t.Fatalf("got %q", got)
	}
}

func TestPathFromURIDecodesEscapes(t *testing.T) {
	got := pathFromURI("file:///tmp/my%20diagram.d2")
	if got != "/tmp/my diagram.d2" {
		t.Fatalf("got %q", got)
	}
}

func TestPathFromURILeavesNonFileURI(t *testing.T) {
	got := pathFromURI("untitled:diagram.d2")
	if got != "untitled:diagram.d2" {
		t.Fatalf("got %q", got)
	}
}

func TestURIFromPathConvertsAbsolutePath(t *testing.T) {
	got := uriFromPath("/tmp/my diagram.d2")
	if got != "file:///tmp/my%20diagram.d2" {
		t.Fatalf("got %q", got)
	}
}

func TestURIFromPathLeavesRelativePath(t *testing.T) {
	got := uriFromPath("index.d2")
	if got != "index.d2" {
		t.Fatalf("got %q", got)
	}
}
