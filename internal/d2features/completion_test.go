package d2features

import "testing"

func TestCompleteReturnsD2Completions(t *testing.T) {
	items, err := Complete("x: { style.", 0, len("x: { style."))
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected completion items")
	}

	for _, item := range items {
		if item.Label == "fill" {
			if item.Kind != completionItemKindProperty {
				t.Fatalf("expected property kind for fill, got %d", item.Kind)
			}
			if item.Detail == "" {
				t.Fatal("expected style detail")
			}
			if item.Documentation == nil || item.Documentation.Value != "`fill` sets the interior color for a shape." {
				t.Fatalf("expected fill documentation, got %#v", item.Documentation)
			}
			return
		}
	}
	t.Fatalf("expected fill completion, got %#v", items)
}

func TestCompleteAddsShapeDocumentation(t *testing.T) {
	items, err := Complete("x.shape: ", 0, len("x.shape: "))
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	for _, item := range items {
		if item.Label == "rectangle" {
			if item.Detail == "" {
				t.Fatal("expected shape detail")
			}
			if item.Documentation == nil || item.Documentation.Value != "`rectangle` is a D2 shape." {
				t.Fatalf("expected rectangle documentation, got %#v", item.Documentation)
			}
			return
		}
	}
	t.Fatalf("expected rectangle completion, got %#v", items)
}
