package d2features

import (
	"strings"
	"testing"
)

func TestRenderSVGReturnsSVG(t *testing.T) {
	svg, err := RenderSVG("index.d2", "x -> y\n", nil)
	if err != nil {
		t.Fatalf("render svg: %v", err)
	}

	if !strings.Contains(svg, "<svg") {
		t.Fatalf("expected svg output, got %q", svg[:min(len(svg), 80)])
	}
}

func TestRenderSVGUsesImportedFiles(t *testing.T) {
	files := map[string]string{
		"index.d2": "hey: @ok\nhey.okay\n",
		"ok.d2":    "okay\n",
	}
	svg, err := RenderSVG("index.d2", files["index.d2"], files)
	if err != nil {
		t.Fatalf("render svg: %v", err)
	}

	if !strings.Contains(svg, "<svg") {
		t.Fatalf("expected svg output, got %q", svg[:min(len(svg), 80)])
	}
}
