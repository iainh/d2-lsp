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

		start, end, err := offsetsForRange(text, *change.Range)
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

func offsetsForRange(text string, r rangePosition) (int, int, error) {
	if r.Start.Line < 0 || r.Start.Character < 0 {
		return 0, 0, fmt.Errorf("invalid negative position line=%d character=%d", r.Start.Line, r.Start.Character)
	}
	if r.End.Line < 0 || r.End.Character < 0 {
		return 0, 0, fmt.Errorf("invalid negative position line=%d character=%d", r.End.Line, r.End.Character)
	}
	if r.Start.Line > r.End.Line || (r.Start.Line == r.End.Line && r.Start.Character > r.End.Character) {
		return 0, 0, fmt.Errorf("invalid content change range: start offset %d is after end offset %d", r.Start.Line, r.End.Line)
	}

	start, end := -1, -1
	line := 0
	character := 0
	if r.Start.Line == 0 && r.Start.Character == 0 {
		start = 0
	}
	if r.End.Line == 0 && r.End.Character == 0 {
		end = 0
	}
	if start >= 0 && end >= 0 {
		return start, end, nil
	}

	for offset, runeValue := range text {
		if runeValue == '\n' {
			line++
			character = 0
			nextOffset := offset + 1
			if start < 0 && r.Start.Line == line && r.Start.Character == character {
				start = nextOffset
			}
			if end < 0 && r.End.Line == line && r.End.Character == character {
				end = nextOffset
			}
			if start >= 0 && end >= 0 {
				return start, end, nil
			}
			continue
		}

		if runeValue > 0xFFFF {
			character += 2
		} else {
			character++
		}
		nextOffset := offset + utf8.RuneLen(runeValue)
		if start < 0 && r.Start.Line == line && r.Start.Character == character {
			start = nextOffset
		}
		if end < 0 && r.End.Line == line && r.End.Character == character {
			end = nextOffset
		}
		if start >= 0 && end >= 0 {
			return start, end, nil
		}
	}

	if start < 0 && r.Start.Line == line && r.Start.Character == character {
		start = len(text)
	}
	if end < 0 && r.End.Line == line && r.End.Character == character {
		end = len(text)
	}
	if start < 0 {
		return 0, 0, fmt.Errorf("position line=%d character=%d is outside document", r.Start.Line, r.Start.Character)
	}
	if end < 0 {
		return 0, 0, fmt.Errorf("position line=%d character=%d is outside document", r.End.Line, r.End.Character)
	}
	return start, end, nil
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
