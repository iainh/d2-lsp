package lsp

import (
	"fmt"
	"unicode/utf8"
)

func applyContentChanges(text string, changes []textDocumentContentChangeEvent) (string, error) {
	for _, change := range changes {
		if change.Range == nil {
			text = change.Text
			continue
		}

		start, err := offsetForPosition(text, change.Range.Start)
		if err != nil {
			return "", err
		}
		end, err := offsetForPosition(text, change.Range.End)
		if err != nil {
			return "", err
		}
		if start > end {
			return "", fmt.Errorf("invalid content change range: start offset %d is after end offset %d", start, end)
		}

		text = text[:start] + change.Text + text[end:]
	}
	return text, nil
}

func offsetForPosition(text string, pos position) (int, error) {
	if pos.Line < 0 || pos.Character < 0 {
		return 0, fmt.Errorf("invalid negative position line=%d character=%d", pos.Line, pos.Character)
	}

	line := 0
	character := 0
	if pos.Line == line && pos.Character == character {
		return 0, nil
	}

	for offset, r := range text {
		if r == '\n' {
			line++
			character = 0
			nextOffset := offset + 1
			if pos.Line == line && pos.Character == character {
				return nextOffset, nil
			}
			continue
		}

		if r > 0xFFFF {
			character += 2
		} else {
			character++
		}
		nextOffset := offset + utf8.RuneLen(r)
		if pos.Line == line && pos.Character == character {
			return nextOffset, nil
		}
	}

	if pos.Line == line && pos.Character == character {
		return len(text), nil
	}
	return 0, fmt.Errorf("position line=%d character=%d is outside document", pos.Line, pos.Character)
}
