package d2diagnostics

import (
	"errors"
	"strings"

	"oss.terrastruct.com/d2/d2ast"
	"oss.terrastruct.com/d2/d2compiler"
	"oss.terrastruct.com/d2/d2parser"
	"oss.terrastruct.com/d2/lib/memfs"
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

const (
	SeverityError = 1
)

func Parse(path, text string) []Diagnostic {
	return parse(path, text, nil)
}

func ParseInFiles(path, text string, files map[string]string) []Diagnostic {
	return parse(path, text, files)
}

func parse(path, text string, files map[string]string) []Diagnostic {
	parseErr := &d2parser.ParseError{}
	_, err := d2parser.Parse(path, strings.NewReader(text), &d2parser.ParseOptions{
		UTF16Pos:   true,
		ParseError: parseErr,
	})
	if err != nil {
		var pe *d2parser.ParseError
		if errors.As(err, &pe) && pe != parseErr {
			parseErr = pe
		}
	}

	if parseErr.Empty() {
		return compileDiagnostics(path, text, files)
	}

	return diagnosticsFromD2Errors(parseErr.Errors)
}

func compileDiagnostics(path, text string, files map[string]string) []Diagnostic {
	options := &d2compiler.CompileOptions{
		UTF16Pos: true,
	}
	if len(files) > 0 {
		fs, err := memfs.New(files)
		if err != nil {
			return []Diagnostic{{
				Range: Range{
					Start: Position{},
					End:   Position{},
				},
				Severity: SeverityError,
				Source:   "d2",
				Message:  err.Error(),
			}}
		}
		options.FS = fs
	}

	_, _, err := d2compiler.Compile(path, strings.NewReader(text), options)
	if err == nil {
		return nil
	}

	var pe *d2parser.ParseError
	if !errors.As(err, &pe) {
		return []Diagnostic{{
			Range: Range{
				Start: Position{},
				End:   Position{},
			},
			Severity: SeverityError,
			Source:   "d2",
			Message:  err.Error(),
		}}
	}

	return diagnosticsFromD2Errors(pe.Errors)
}

func diagnosticsFromD2Errors(errs []d2ast.Error) []Diagnostic {
	diagnostics := make([]Diagnostic, 0, len(errs))
	for _, err := range errs {
		diagnostics = append(diagnostics, fromD2Error(err))
	}
	return diagnostics
}

func fromD2Error(err d2ast.Error) Diagnostic {
	return Diagnostic{
		Range: Range{
			Start: fromD2Position(err.Range.Start),
			End:   fromD2Position(err.Range.End),
		},
		Severity: SeverityError,
		Source:   "d2",
		Message:  err.Message,
	}
}

func fromD2Position(pos d2ast.Position) Position {
	return Position{
		Line:      nonnegative(pos.Line),
		Character: nonnegative(pos.Column),
	}
}

func nonnegative(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
