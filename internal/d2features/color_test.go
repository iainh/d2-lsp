package d2features

import "testing"

func TestDocumentColorsReturnsHexColorValues(t *testing.T) {
	colors, err := DocumentColors("file:///diagram.d2", "x.style.fill: '#ff0000'\n")
	if err != nil {
		t.Fatalf("document colors: %v", err)
	}
	if len(colors) != 1 {
		t.Fatalf("expected one color, got %#v", colors)
	}
	if colors[0].Color.Red != 1 || colors[0].Color.Green != 0 || colors[0].Color.Blue != 0 || colors[0].Color.Alpha != 1 {
		t.Fatalf("unexpected color %#v", colors[0].Color)
	}
	if colors[0].Range.Start.Line != 0 || colors[0].Range.Start.Character != len("x.style.fill: ") {
		t.Fatalf("unexpected range %#v", colors[0].Range)
	}
}

func TestDocumentColorsReturnsNamedColorValues(t *testing.T) {
	colors, err := DocumentColors("file:///diagram.d2", "x: {style: {stroke: blue}}\n")
	if err != nil {
		t.Fatalf("document colors: %v", err)
	}
	if len(colors) != 1 {
		t.Fatalf("expected one color, got %#v", colors)
	}
	if colors[0].Color.Blue != 1 {
		t.Fatalf("unexpected color %#v", colors[0].Color)
	}
}

func TestDocumentColorsIgnoresNonColorStyleValues(t *testing.T) {
	colors, err := DocumentColors("file:///diagram.d2", "x.style.fill-pattern: dots\nx.style.fill: N1\n")
	if err != nil {
		t.Fatalf("document colors: %v", err)
	}
	if len(colors) != 0 {
		t.Fatalf("expected no colors, got %#v", colors)
	}
}

func TestDocumentColorsToleratesInvalidDocument(t *testing.T) {
	colors, err := DocumentColors("file:///diagram.d2", "x: {style.fill: red\n")
	if err != nil {
		t.Fatalf("document colors: %v", err)
	}
	if len(colors) > 1 {
		t.Fatalf("unexpected colors %#v", colors)
	}
}

func TestColorPresentationsReturnsHexLabel(t *testing.T) {
	r := Range{Start: Position{Line: 1, Character: 2}, End: Position{Line: 1, Character: 9}}
	presentations := ColorPresentations(Color{Red: 1, Green: 0.5, Blue: 0, Alpha: 1}, r)
	if len(presentations) != 1 {
		t.Fatalf("expected one presentation, got %#v", presentations)
	}
	if presentations[0].Label != "#ff8000" {
		t.Fatalf("unexpected label %q", presentations[0].Label)
	}
	if presentations[0].TextEdit.NewText != "#ff8000" {
		t.Fatalf("unexpected text edit %#v", presentations[0].TextEdit)
	}
	if presentations[0].TextEdit.Range != r {
		t.Fatalf("unexpected text edit range %#v", presentations[0].TextEdit.Range)
	}
}

func TestColorPresentationsIncludesAlphaWhenTransparent(t *testing.T) {
	presentations := ColorPresentations(Color{Red: 1, Green: 0, Blue: 0, Alpha: 0.5}, Range{})
	if presentations[0].Label != "#ff000080" {
		t.Fatalf("unexpected label %q", presentations[0].Label)
	}
}
