package d2features

import (
	"errors"
	"strings"

	"oss.terrastruct.com/d2/d2format"
	"oss.terrastruct.com/d2/d2parser"
)

func Format(text string) (string, bool, error) {
	parseErr := &d2parser.ParseError{}
	ast, err := d2parser.Parse("", strings.NewReader(text), &d2parser.ParseOptions{
		ParseError: parseErr,
	})
	if err != nil {
		var pe *d2parser.ParseError
		if errors.As(err, &pe) {
			return "", false, nil
		}
		return "", false, err
	}
	if !parseErr.Empty() {
		return "", false, nil
	}

	formatted := d2format.Format(ast)
	if formatted != "" && !strings.HasSuffix(formatted, "\n") {
		formatted += "\n"
	}
	return formatted, formatted != text, nil
}
