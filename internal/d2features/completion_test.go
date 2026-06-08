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
			return
		}
	}
	t.Fatalf("expected fill completion, got %#v", items)
}
