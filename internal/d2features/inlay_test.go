package d2features

import "testing"

func TestInlayHintsReturnResolvedImportPaths(t *testing.T) {
	hints, err := InlayHints("/workspace/index.d2", "hey: @ok\n")
	if err != nil {
		t.Fatalf("inlay hints: %v", err)
	}

	if len(hints) != 1 {
		t.Fatalf("expected one hint, got %#v", hints)
	}
	if hints[0].Label != " => /workspace/ok.d2" {
		t.Fatalf("unexpected hint label %q", hints[0].Label)
	}
	if hints[0].Position.Line != 0 || hints[0].Position.Character != len("hey: @ok") {
		t.Fatalf("unexpected hint position %#v", hints[0].Position)
	}
}

func TestInlayHintsReturnSpreadImportPaths(t *testing.T) {
	hints, err := InlayHints("/workspace/index.d2", "...@models.user\n")
	if err != nil {
		t.Fatalf("inlay hints: %v", err)
	}

	if len(hints) != 1 {
		t.Fatalf("expected one hint, got %#v", hints)
	}
	if hints[0].Label != " => /workspace/models/user.d2" {
		t.Fatalf("unexpected hint label %q", hints[0].Label)
	}
}

func TestInlayHintsReturnEmptyForInvalidDocument(t *testing.T) {
	hints, err := InlayHints("/workspace/index.d2", "x: {\n")
	if err != nil {
		t.Fatalf("inlay hints: %v", err)
	}

	if len(hints) != 0 {
		t.Fatalf("expected no hints, got %#v", hints)
	}
}
