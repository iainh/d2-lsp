package d2features

import (
	"errors"
	"fmt"
	"strings"

	"oss.terrastruct.com/d2/d2parser"
)

func ValidateRenameName(name string) error {
	if name == "" {
		return fmt.Errorf("rename target cannot be empty")
	}
	if strings.TrimSpace(name) != name {
		return fmt.Errorf("rename target cannot start or end with whitespace")
	}
	if strings.ContainsAny(name, "\r\n") {
		return fmt.Errorf("rename target cannot contain newlines")
	}

	ast, err := d2parser.Parse("rename.d2", strings.NewReader(name+"\n"), &d2parser.ParseOptions{
		UTF16Pos: true,
	})
	if err != nil {
		var pe *d2parser.ParseError
		if errors.As(err, &pe) {
			return fmt.Errorf("rename target must be valid D2 syntax")
		}
		return err
	}
	if ast == nil || len(ast.Nodes) != 1 || ast.Nodes[0].MapKey == nil {
		return fmt.Errorf("rename target must be a single D2 key or edge")
	}

	mk := ast.Nodes[0].MapKey
	if mk.Value.Unbox() != nil || mk.Value.Map != nil || mk.Value.Import != nil {
		return fmt.Errorf("rename target cannot include a value")
	}
	if symbolName(mk) != name {
		return fmt.Errorf("rename target must be a single D2 key or edge")
	}
	return nil
}
