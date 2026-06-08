package d2features

import "testing"

func TestDocumentLinksReturnsLinkValues(t *testing.T) {
	links, err := DocumentLinks("file:///diagram.d2", "x: {link: https://example.com}\n")
	if err != nil {
		t.Fatalf("document links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected one link, got %#v", links)
	}
	if links[0].Target != "https://example.com" {
		t.Fatalf("unexpected target %q", links[0].Target)
	}
	if links[0].Range.Start.Line != 0 || links[0].Range.Start.Character != len("x: {link: ") {
		t.Fatalf("unexpected range %#v", links[0].Range)
	}
}

func TestDocumentLinksReturnsIconValues(t *testing.T) {
	links, err := DocumentLinks("file:///diagram.d2", "x.icon: https://example.com/icon.svg\n")
	if err != nil {
		t.Fatalf("document links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected one link, got %#v", links)
	}
	if links[0].Target != "https://example.com/icon.svg" {
		t.Fatalf("unexpected target %q", links[0].Target)
	}
}

func TestDocumentLinksIgnoresNonURLValues(t *testing.T) {
	links, err := DocumentLinks("file:///diagram.d2", "x: {link: internal-id}\n")
	if err != nil {
		t.Fatalf("document links: %v", err)
	}
	if links == nil {
		t.Fatal("expected empty links slice, got nil")
	}
	if len(links) != 0 {
		t.Fatalf("expected no links, got %#v", links)
	}
}

func TestDocumentLinksToleratesInvalidDocument(t *testing.T) {
	links, err := DocumentLinks("file:///diagram.d2", "x: {link: https://example.com\n")
	if err != nil {
		t.Fatalf("document links: %v", err)
	}
	if links == nil {
		t.Fatal("expected links slice, got nil")
	}
	if len(links) > 1 {
		t.Fatalf("unexpected links %#v", links)
	}
}
