package d2diagnostics

import (
	"errors"
	"io/fs"
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

func ParseAllInFiles(files map[string]string) map[string][]Diagnostic {
	diagnostics := make(map[string][]Diagnostic, len(files))
	fileSystem, err := memfs.New(files)
	if err != nil {
		for path := range files {
			diagnostics[path] = []Diagnostic{errorDiagnostic(err)}
		}
		return diagnostics
	}

	for path, text := range files {
		diagnostics[path] = compileDiagnosticsWithFS(path, text, fileSystem)
	}
	return diagnostics
}

func parse(path, text string, files map[string]string) []Diagnostic {
	return compileDiagnostics(path, text, files)
}

func compileDiagnostics(path, text string, files map[string]string) []Diagnostic {
	var fileSystem fs.FS
	if len(files) > 0 {
		memFileSystem, err := memfs.New(files)
		if err != nil {
			return []Diagnostic{errorDiagnostic(err)}
		}
		fileSystem = memFileSystem
	}
	return compileDiagnosticsWithFS(path, text, fileSystem)
}

func compileDiagnosticsWithFS(path, text string, fs fs.FS) []Diagnostic {
	options := &d2compiler.CompileOptions{
		UTF16Pos: true,
		FS:       fs,
	}
	_, _, err := d2compiler.Compile(path, strings.NewReader(text), options)
	if err == nil {
		return nil
	}

	var pe *d2parser.ParseError
	if !errors.As(err, &pe) {
		return []Diagnostic{errorDiagnostic(err)}
	}

	return diagnosticsFromD2Errors(pe.Errors)
}

func errorDiagnostic(err error) Diagnostic {
	return Diagnostic{
		Range: Range{
			Start: Position{},
			End:   Position{},
		},
		Severity: SeverityError,
		Source:   "d2",
		Message:  err.Error(),
	}
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
