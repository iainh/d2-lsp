package d2features

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkSymbolsLargeDocument(b *testing.B) {
	text := benchmarkD2Document(1000)
	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		symbols, err := Symbols("/workspace/diagram.d2", text)
		if err != nil {
			b.Fatal(err)
		}
		if len(symbols) == 0 {
			b.Fatal("expected symbols")
		}
	}
}

func BenchmarkSemanticTokensLargeDocument(b *testing.B) {
	text := benchmarkD2Document(1000)
	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		tokens, err := SemanticTokensFor("/workspace/diagram.d2", text)
		if err != nil {
			b.Fatal(err)
		}
		if len(tokens.Data) == 0 {
			b.Fatal("expected semantic tokens")
		}
	}
}

func BenchmarkInlayHintsLargeDocument(b *testing.B) {
	text := benchmarkD2Imports(1000)
	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		hints, err := InlayHints("/workspace/diagram.d2", text)
		if err != nil {
			b.Fatal(err)
		}
		if len(hints) == 0 {
			b.Fatal("expected inlay hints")
		}
	}
}

func BenchmarkFormatLargeDocument(b *testing.B) {
	text := benchmarkD2UnformattedDocument(1000)
	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		formatted, changed, err := Format(text)
		if err != nil {
			b.Fatal(err)
		}
		if !changed || formatted == "" {
			b.Fatal("expected formatted output")
		}
	}
}

func BenchmarkCompleteLargeDocument(b *testing.B) {
	text := benchmarkD2Document(1000) + "\nshape: "
	line := strings.Count(text, "\n")
	character := len("shape: ")
	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		items, err := Complete(text, line, character)
		if err != nil {
			b.Fatal(err)
		}
		if len(items) == 0 {
			b.Fatal("expected completions")
		}
	}
}

func benchmarkD2Document(nodes int) string {
	var b strings.Builder
	b.Grow(nodes * 96)
	for i := 0; i < nodes; i++ {
		fmt.Fprintf(&b, "service_%04d: {\n", i)
		fmt.Fprintf(&b, "  label: \"Service %04d\"\n", i)
		b.WriteString("  shape: rectangle\n")
		fmt.Fprintf(&b, "  style.fill: \"#%06x\"\n", i%0xffffff)
		b.WriteString("}\n")
		if i > 0 {
			fmt.Fprintf(&b, "service_%04d -> service_%04d\n", i-1, i)
		}
	}
	return b.String()
}

func benchmarkD2Imports(imports int) string {
	var b strings.Builder
	b.Grow(imports * 40)
	for i := 0; i < imports; i++ {
		fmt.Fprintf(&b, "service_%04d: @models.service_%04d\n", i, i)
	}
	return b.String()
}

func benchmarkD2UnformattedDocument(nodes int) string {
	var b strings.Builder
	b.Grow(nodes * 32)
	for i := 0; i < nodes; i++ {
		fmt.Fprintf(&b, "service_%04d:{shape:rectangle}\n", i)
	}
	return b.String()
}
