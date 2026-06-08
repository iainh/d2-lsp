package d2features

import "testing"

func TestSymbolsReturnsNestedDocumentSymbols(t *testing.T) {
	symbols, err := Symbols("file:///diagram.d2", "server: {\n  shape: rectangle\n}\nserver -> database\n")
	if err != nil {
		t.Fatalf("symbols: %v", err)
	}
	if len(symbols) != 2 {
		t.Fatalf("expected two top-level symbols, got %#v", symbols)
	}

	if symbols[0].Name != "server" {
		t.Fatalf("unexpected first symbol name %q", symbols[0].Name)
	}
	if symbols[0].Kind != symbolKindObject {
		t.Fatalf("expected object symbol, got %d", symbols[0].Kind)
	}
	if len(symbols[0].Children) != 1 {
		t.Fatalf("expected nested symbol, got %#v", symbols[0].Children)
	}
	if symbols[0].Children[0].Name != "shape" {
		t.Fatalf("unexpected child symbol name %q", symbols[0].Children[0].Name)
	}
	if symbols[1].Name != "server -> database" {
		t.Fatalf("unexpected edge symbol name %q", symbols[1].Name)
	}
}

func TestSymbolsUsesUTF16Positions(t *testing.T) {
	symbols, err := Symbols("file:///diagram.d2", "🙂server: ok\n")
	if err != nil {
		t.Fatalf("symbols: %v", err)
	}
	if len(symbols) != 1 {
		t.Fatalf("expected one symbol, got %#v", symbols)
	}
	if symbols[0].SelectionRange.End.Character <= len("server") {
		t.Fatalf("expected UTF-16 character offset to include emoji width, got %#v", symbols[0].SelectionRange)
	}
}

func TestSymbolsIncludeSpreadImports(t *testing.T) {
	symbols, err := Symbols("file:///diagram.d2", "...@models.user\n")
	if err != nil {
		t.Fatalf("symbols: %v", err)
	}
	if len(symbols) != 1 {
		t.Fatalf("expected one symbol, got %#v", symbols)
	}
	if symbols[0].Name != "...@models.user" {
		t.Fatalf("unexpected import symbol name %q", symbols[0].Name)
	}
	if symbols[0].Kind != symbolKindFile {
		t.Fatalf("expected file symbol, got %d", symbols[0].Kind)
	}
}

func TestSymbolsIncludeValueImportsAsChildren(t *testing.T) {
	symbols, err := Symbols("file:///diagram.d2", "service: @models.user\n")
	if err != nil {
		t.Fatalf("symbols: %v", err)
	}
	if len(symbols) != 1 {
		t.Fatalf("expected one symbol, got %#v", symbols)
	}
	if len(symbols[0].Children) != 1 {
		t.Fatalf("expected import child symbol, got %#v", symbols[0].Children)
	}
	child := symbols[0].Children[0]
	if child.Name != "@models.user" {
		t.Fatalf("unexpected import child name %q", child.Name)
	}
	if child.Kind != symbolKindFile {
		t.Fatalf("expected file child symbol, got %d", child.Kind)
	}
}
