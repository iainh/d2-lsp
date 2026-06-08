package d2diagnostics

import "testing"

func TestParseReturnsNoDiagnosticsForValidD2(t *testing.T) {
	diagnostics := Parse("file:///diagram.d2", "x -> y\n")
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
}

func TestParseReturnsDiagnosticsForInvalidD2(t *testing.T) {
	diagnostics := Parse("file:///diagram.d2", "x: {\n")
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}

	got := diagnostics[0]
	if got.Source != "d2" {
		t.Fatalf("expected d2 source, got %q", got.Source)
	}
	if got.Severity != SeverityError {
		t.Fatalf("expected error severity, got %d", got.Severity)
	}
	if got.Message == "" {
		t.Fatal("expected diagnostic message")
	}
}

func TestParseReturnsCompilerDiagnosticsForSemanticD2Errors(t *testing.T) {
	diagnostics := Parse("file:///diagram.d2", "x: {shape: not-a-shape}\n")
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}

	got := diagnostics[0]
	if got.Source != "d2" {
		t.Fatalf("expected d2 source, got %q", got.Source)
	}
	if got.Severity != SeverityError {
		t.Fatalf("expected error severity, got %d", got.Severity)
	}
	if got.Message == "" {
		t.Fatal("expected diagnostic message")
	}
}

func TestParseInFilesUsesImportedFiles(t *testing.T) {
	files := map[string]string{
		"index.d2": "hey: @ok\nhey.okay\n",
		"ok.d2":    "okay\n",
	}

	diagnostics := ParseInFiles("index.d2", files["index.d2"], files)
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
}

func TestParseInFilesReturnsImportedCompilerDiagnostics(t *testing.T) {
	files := map[string]string{
		"index.d2": "...@ok\n",
		"ok.d2":    "x: {shape: not-a-shape}\n",
	}

	diagnostics := ParseInFiles("index.d2", files["index.d2"], files)
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
	if diagnostics[0].Message == "" {
		t.Fatal("expected diagnostic message")
	}
}

func TestParseAllInFilesUsesImportedFiles(t *testing.T) {
	files := map[string]string{
		"index.d2": "hey: @ok\nhey.okay\n",
		"ok.d2":    "okay\n",
	}

	diagnostics := ParseAllInFiles(files)
	if len(diagnostics) != len(files) {
		t.Fatalf("expected diagnostics for %d files, got %d", len(files), len(diagnostics))
	}
	for path, pathDiagnostics := range diagnostics {
		if len(pathDiagnostics) != 0 {
			t.Fatalf("expected no diagnostics for %s, got %#v", path, pathDiagnostics)
		}
	}
}
