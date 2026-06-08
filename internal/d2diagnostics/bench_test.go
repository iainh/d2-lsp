package d2diagnostics

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkParseLargeDocument(b *testing.B) {
	text := benchmarkD2Document(1000)
	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		diagnostics := Parse("/workspace/diagram.d2", text)
		if len(diagnostics) != 0 {
			b.Fatalf("expected no diagnostics, got %d", len(diagnostics))
		}
	}
}

func BenchmarkParseInFilesWithImports(b *testing.B) {
	text := benchmarkD2Imports(250)
	files := map[string]string{
		"diagram.d2": text,
	}
	for i := 0; i < 250; i++ {
		files[fmt.Sprintf("service_%04d.d2", i)] = fmt.Sprintf("model_%04d\n", i)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		diagnostics := ParseInFiles("diagram.d2", text, files)
		if len(diagnostics) != 0 {
			b.Fatalf("expected no diagnostics, got %d; first: %#v", len(diagnostics), diagnostics[0])
		}
	}
}

func BenchmarkParseAllInFilesWithImports(b *testing.B) {
	files := benchmarkD2WorkspaceImports(250)

	b.ReportAllocs()
	for b.Loop() {
		diagnostics := ParseAllInFiles(files)
		if len(diagnostics) != len(files) {
			b.Fatalf("expected diagnostics for %d files, got %d", len(files), len(diagnostics))
		}
		for path, pathDiagnostics := range diagnostics {
			if len(pathDiagnostics) != 0 {
				b.Fatalf("expected no diagnostics for %s, got %d; first: %#v", path, len(pathDiagnostics), pathDiagnostics[0])
			}
		}
	}
}

func BenchmarkParseWorkspaceInFilesRepeatedMemFS(b *testing.B) {
	files := benchmarkD2WorkspaceImports(250)

	b.ReportAllocs()
	for b.Loop() {
		for path, text := range files {
			diagnostics := ParseInFiles(path, text, files)
			if len(diagnostics) != 0 {
				b.Fatalf("expected no diagnostics for %s, got %d; first: %#v", path, len(diagnostics), diagnostics[0])
			}
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
	b.Grow(imports * 64)
	for i := 0; i < imports; i++ {
		fmt.Fprintf(&b, "service_%04d: @service_%04d\n", i, i)
		fmt.Fprintf(&b, "service_%04d.model_%04d\n", i, i)
	}
	return b.String()
}

func benchmarkD2WorkspaceImports(imports int) map[string]string {
	files := map[string]string{}
	for i := 0; i < imports; i++ {
		path := fmt.Sprintf("diagram_%04d.d2", i)
		files[path] = fmt.Sprintf("service_%04d: @service_%04d\nservice_%04d.model_%04d\n", i, i, i, i)
		files[fmt.Sprintf("service_%04d.d2", i)] = fmt.Sprintf("model_%04d\n", i)
	}
	return files
}
