package d2features

import "testing"

func TestHoverAtReturnsKeywordHover(t *testing.T) {
	hover, err := HoverAt("file:///diagram.d2", "x: {shape: rectangle}\n", 0, 4)
	if err != nil {
		t.Fatalf("hover: %v", err)
	}
	if hover == nil {
		t.Fatal("expected hover")
	}
	if hover.Contents.Kind != markupKindMarkdown {
		t.Fatalf("unexpected markup kind %q", hover.Contents.Kind)
	}
	if hover.Contents.Value == "" {
		t.Fatal("expected hover content")
	}
}

func TestHoverAtReturnsShapeValueHover(t *testing.T) {
	hover, err := HoverAt("file:///diagram.d2", "x: {shape: rectangle}\n", 0, len("x: {shape: rec"))
	if err != nil {
		t.Fatalf("hover: %v", err)
	}
	if hover == nil {
		t.Fatal("expected hover")
	}
	if hover.Contents.Value != "`rectangle` is a D2 shape." {
		t.Fatalf("unexpected hover content %q", hover.Contents.Value)
	}
}

func TestHoverAtReturnsNilWithoutKnownHover(t *testing.T) {
	hover, err := HoverAt("file:///diagram.d2", "x -> y\n", 0, 0)
	if err != nil {
		t.Fatalf("hover: %v", err)
	}
	if hover != nil {
		t.Fatalf("expected no hover, got %#v", hover)
	}
}
