package d2features

import "testing"

func TestValidateRenameNameAcceptsKeysAndEdges(t *testing.T) {
	for _, name := range []string{"server", "system.api", "source -> target", `"quoted name"`} {
		if err := ValidateRenameName(name); err != nil {
			t.Fatalf("expected %q to be valid: %v", name, err)
		}
	}
}

func TestValidateRenameNameRejectsInvalidNames(t *testing.T) {
	for _, name := range []string{"", " server", "server\napi", "server: api", "x: {y: z}"} {
		if err := ValidateRenameName(name); err == nil {
			t.Fatalf("expected %q to be invalid", name)
		}
	}
}
