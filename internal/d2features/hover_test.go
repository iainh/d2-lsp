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

func TestHoverAtReturnsSpecificKeywordHover(t *testing.T) {
	hover, err := HoverAt("file:///diagram.d2", "x: {link: https://example.com}\n", 0, len("x: {li"))
	if err != nil {
		t.Fatalf("hover: %v", err)
	}
	if hover == nil {
		t.Fatal("expected hover")
	}
	if hover.Contents.Value != "`link` attaches a URL or board path to a shape or connection." {
		t.Fatalf("unexpected hover content %q", hover.Contents.Value)
	}
}

func TestHoverAtReturnsSpecificStyleHover(t *testing.T) {
	hover, err := HoverAt("file:///diagram.d2", "x.style.opacity: 0.5\n", 0, len("x.style.opa"))
	if err != nil {
		t.Fatalf("hover: %v", err)
	}
	if hover == nil {
		t.Fatal("expected hover")
	}
	if hover.Contents.Value != "`opacity` sets transparency from 0.0 to 1.0." {
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
