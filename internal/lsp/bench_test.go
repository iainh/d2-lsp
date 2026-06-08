package lsp

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkApplyContentChangesAppendLargeDocument(b *testing.B) {
	text := benchmarkTextDocument(10000)
	line := strings.Count(text, "\n")
	changes := []textDocumentContentChangeEvent{{
		Range: &rangePosition{
			Start: position{Line: line, Character: 0},
			End:   position{Line: line, Character: 0},
		},
		Text: "tail -> next\n",
	}}

	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		got, err := applyContentChanges(text, changes)
		if err != nil {
			b.Fatal(err)
		}
		if len(got) <= len(text) {
			b.Fatal("expected appended text")
		}
	}
}

func BenchmarkApplyContentChangesEditMiddleLargeDocument(b *testing.B) {
	text := benchmarkTextDocument(10000)
	line := 5000
	changes := []textDocumentContentChangeEvent{{
		Range: &rangePosition{
			Start: position{Line: line, Character: len("node_5000 -> ")},
			End:   position{Line: line, Character: len("node_5000 -> node_5001")},
		},
		Text: "replacement",
	}}

	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		got, err := applyContentChanges(text, changes)
		if err != nil {
			b.Fatal(err)
		}
		if len(got) == len(text) {
			b.Fatal("expected changed text length")
		}
	}
}

func benchmarkTextDocument(lines int) string {
	var b strings.Builder
	b.Grow(lines * 24)
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&b, "node_%04d -> node_%04d\n", i, i+1)
	}
	return b.String()
}
