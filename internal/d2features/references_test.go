package d2features

import "testing"

func TestReferencesReturnsD2RefRangesForObject(t *testing.T) {
	text := "x\nx -> y\n"
	locations, err := References("file:///diagram.d2", text, 1, 0, true)
	if err != nil {
		t.Fatalf("references: %v", err)
	}
	if len(locations) != 2 {
		t.Fatalf("expected two locations, got %#v", locations)
	}
	if locations[0].Range.Start.Line != 0 {
		t.Fatalf("expected declaration on line 0, got %#v", locations[0])
	}
	if locations[1].Range.Start.Line != 1 {
		t.Fatalf("expected edge reference on line 1, got %#v", locations[1])
	}
}

func TestReferencesCanExcludeDeclaration(t *testing.T) {
	text := "x\nx -> y\n"
	locations, err := References("file:///diagram.d2", text, 1, 0, false)
	if err != nil {
		t.Fatalf("references: %v", err)
	}
	if len(locations) != 1 {
		t.Fatalf("expected one location, got %#v", locations)
	}
	if locations[0].Range.Start.Line != 1 {
		t.Fatalf("expected only reference line, got %#v", locations[0])
	}
}

func TestReferencesReturnsEmptyWhenNoKeyAtPosition(t *testing.T) {
	locations, err := References("file:///diagram.d2", "x\n\n", 1, 0, true)
	if err != nil {
		t.Fatalf("references: %v", err)
	}
	if locations == nil {
		t.Fatal("expected empty locations slice, got nil")
	}
	if len(locations) != 0 {
		t.Fatalf("expected no locations, got %#v", locations)
	}
}

func TestDefinitionReturnsFirstD2ReferenceRange(t *testing.T) {
	text := "x\nx -> y\n"
	location, err := Definition("file:///diagram.d2", text, 1, 0)
	if err != nil {
		t.Fatalf("definition: %v", err)
	}
	if location == nil {
		t.Fatal("expected definition location")
	}
	if location.Range.Start.Line != 0 {
		t.Fatalf("expected definition on line 0, got %#v", location)
	}
}

func TestDefinitionInFilesReturnsImportedReferenceURI(t *testing.T) {
	fs := map[string]string{
		"index.d2": "hey: @ok\nhey.okay\n",
		"ok.d2":    "okay\n",
	}
	uriByPath := map[string]string{
		"index.d2": "file:///workspace/index.d2",
		"ok.d2":    "file:///workspace/ok.d2",
	}

	location, err := DefinitionInFiles("file:///workspace/index.d2", "index.d2", fs, uriByPath, 1, len("hey.ok"))
	if err != nil {
		t.Fatalf("definition: %v", err)
	}
	if location == nil {
		t.Fatal("expected definition location")
	}
	if location.URI != "file:///workspace/ok.d2" {
		t.Fatalf("unexpected uri %q", location.URI)
	}
	if location.Range.Start.Line != 0 {
		t.Fatalf("expected imported definition on line 0, got %#v", location)
	}
}

func TestDefinitionReturnsNilWhenNoKeyAtPosition(t *testing.T) {
	location, err := Definition("file:///diagram.d2", "x\n\n", 1, 0)
	if err != nil {
		t.Fatalf("definition: %v", err)
	}
	if location != nil {
		t.Fatalf("expected no definition, got %#v", location)
	}
}

func TestDocumentHighlightsReturnsLocalReferenceRanges(t *testing.T) {
	text := "x\nx -> y\n"
	highlights, err := DocumentHighlights("file:///diagram.d2", text, 1, 0)
	if err != nil {
		t.Fatalf("document highlights: %v", err)
	}
	if len(highlights) != 2 {
		t.Fatalf("expected two highlights, got %#v", highlights)
	}
	if highlights[0].Kind != documentHighlightKindText {
		t.Fatalf("unexpected highlight kind %d", highlights[0].Kind)
	}
	if highlights[0].Range.Start.Line != 0 || highlights[1].Range.Start.Line != 1 {
		t.Fatalf("unexpected highlights %#v", highlights)
	}
}

func TestDocumentHighlightsReturnsEmptyWhenNoKeyAtPosition(t *testing.T) {
	highlights, err := DocumentHighlights("file:///diagram.d2", "x\n\n", 1, 0)
	if err != nil {
		t.Fatalf("document highlights: %v", err)
	}
	if len(highlights) != 0 {
		t.Fatalf("expected no highlights, got %#v", highlights)
	}
}
