package lsp

import "testing"

func TestApplyContentChangesAppliesFullReplacement(t *testing.T) {
	got, err := applyContentChanges("a -> b\n", []textDocumentContentChangeEvent{
		{Text: "x -> y\n"},
	})
	if err != nil {
		t.Fatalf("apply changes: %v", err)
	}
	if got != "x -> y\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyContentChangesAppliesIncrementalRangeReplacement(t *testing.T) {
	got, err := applyContentChanges("a -> b\n", []textDocumentContentChangeEvent{
		{
			Range: &rangePosition{
				Start: position{Line: 0, Character: len("a -> ")},
				End:   position{Line: 0, Character: len("a -> b")},
			},
			Text: "c",
		},
	})
	if err != nil {
		t.Fatalf("apply changes: %v", err)
	}
	if got != "a -> c\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyContentChangesUsesUTF16Positions(t *testing.T) {
	got, err := applyContentChanges("a🙂 -> b\n", []textDocumentContentChangeEvent{
		{
			Range: &rangePosition{
				Start: position{Line: 0, Character: 7},
				End:   position{Line: 0, Character: 8},
			},
			Text: "c",
		},
	})
	if err != nil {
		t.Fatalf("apply changes: %v", err)
	}
	if got != "a🙂 -> c\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyContentChangesRejectsPositionInsideUTF16SurrogatePair(t *testing.T) {
	_, err := applyContentChanges("a🙂 -> b\n", []textDocumentContentChangeEvent{
		{
			Range: &rangePosition{
				Start: position{Line: 0, Character: 2},
				End:   position{Line: 0, Character: 3},
			},
			Text: "x",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyContentChangesAppliesMultipleChangesInOrder(t *testing.T) {
	got, err := applyContentChanges("a -> b\n", []textDocumentContentChangeEvent{
		{
			Range: &rangePosition{
				Start: position{Line: 0, Character: 0},
				End:   position{Line: 0, Character: 1},
			},
			Text: "x",
		},
		{
			Range: &rangePosition{
				Start: position{Line: 0, Character: len("x -> ")},
				End:   position{Line: 0, Character: len("x -> b")},
			},
			Text: "y",
		},
	})
	if err != nil {
		t.Fatalf("apply changes: %v", err)
	}
	if got != "x -> y\n" {
		t.Fatalf("got %q", got)
	}
}
