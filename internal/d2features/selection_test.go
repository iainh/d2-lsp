package d2features

import "testing"

func TestSelectionRangesReturnsSmallestContainingRangeFirst(t *testing.T) {
	ranges, err := SelectionRanges("file:///diagram.d2", "server: {shape: rectangle}\n", []Position{{
		Line:      0,
		Character: len("server: {shape: rec"),
	}})
	if err != nil {
		t.Fatalf("selection ranges: %v", err)
	}
	if len(ranges) != 1 {
		t.Fatalf("expected one range, got %#v", ranges)
	}
	if ranges[0] == nil {
		t.Fatal("expected selection range")
	}
	if ranges[0].Range.Start.Line != 0 || ranges[0].Range.Start.Character != len("server: {shape: ") {
		t.Fatalf("expected rectangle scalar range first, got %#v", ranges[0].Range)
	}
	if ranges[0].Range.End.Character != len("server: {shape: rectangle") {
		t.Fatalf("unexpected scalar range %#v", ranges[0].Range)
	}
	if ranges[0].Parent == nil {
		t.Fatal("expected parent selection range")
	}
	if ranges[0].Parent.Range.Start.Character >= ranges[0].Range.Start.Character &&
		ranges[0].Parent.Range.End.Character <= ranges[0].Range.End.Character {
		t.Fatalf("expected parent to be larger, got child %#v parent %#v", ranges[0].Range, ranges[0].Parent.Range)
	}
}

func TestSelectionRangesReturnsOneResultPerPosition(t *testing.T) {
	ranges, err := SelectionRanges("file:///diagram.d2", "a\nb\n", []Position{
		{Line: 0, Character: 0},
		{Line: 1, Character: 0},
	})
	if err != nil {
		t.Fatalf("selection ranges: %v", err)
	}
	if len(ranges) != 2 {
		t.Fatalf("expected two ranges, got %#v", ranges)
	}
	if ranges[0] == nil || ranges[1] == nil {
		t.Fatalf("expected both ranges, got %#v", ranges)
	}
	if ranges[0].Range.Start.Line != 0 || ranges[1].Range.Start.Line != 1 {
		t.Fatalf("unexpected ranges %#v", ranges)
	}
}

func TestSelectionRangesKeepsResultShapeForInvalidDocument(t *testing.T) {
	ranges, err := SelectionRanges("file:///diagram.d2", "x: {\n", []Position{{Line: 0, Character: 0}})
	if err != nil {
		t.Fatalf("selection ranges: %v", err)
	}
	if len(ranges) != 1 {
		t.Fatalf("expected one result, got %#v", ranges)
	}
}
