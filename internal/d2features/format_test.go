package d2features

import "testing"

func TestFormatFormatsValidD2(t *testing.T) {
	formatted, changed, err := Format("x:{y:z}\n")
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	if !changed {
		t.Fatal("expected formatting change")
	}

	want := "x: {y: z}\n"
	if formatted != want {
		t.Fatalf("got %q, want %q", formatted, want)
	}
}

func TestFormatDoesNotFormatInvalidD2(t *testing.T) {
	formatted, changed, err := Format("x: {\n")
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	if changed {
		t.Fatalf("expected no change, got %q", formatted)
	}
}
