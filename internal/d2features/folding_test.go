package d2features

import "testing"

func TestFoldingRangesReturnsNestedMapRanges(t *testing.T) {
	ranges, err := FoldingRanges("file:///diagram.d2", "server: {\n  api: {\n    shape: rectangle\n  }\n}\n")
	if err != nil {
		t.Fatalf("folding ranges: %v", err)
	}
	if len(ranges) != 2 {
		t.Fatalf("expected two folding ranges, got %#v", ranges)
	}

	if ranges[0].StartLine != 0 || ranges[0].EndLine != 4 {
		t.Fatalf("unexpected outer range %#v", ranges[0])
	}
	if ranges[1].StartLine != 1 || ranges[1].EndLine != 3 {
		t.Fatalf("unexpected inner range %#v", ranges[1])
	}
	if ranges[0].Kind != foldingRangeKindRegion {
		t.Fatalf("unexpected folding kind %q", ranges[0].Kind)
	}
}

func TestFoldingRangesSkipsSingleLineMaps(t *testing.T) {
	ranges, err := FoldingRanges("file:///diagram.d2", "server: {shape: rectangle}\n")
	if err != nil {
		t.Fatalf("folding ranges: %v", err)
	}
	if len(ranges) != 0 {
		t.Fatalf("expected no folding ranges, got %#v", ranges)
	}
}
