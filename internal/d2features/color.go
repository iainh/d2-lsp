package d2features

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/mazznoer/csscolorparser"
	"oss.terrastruct.com/d2/d2ast"
	"oss.terrastruct.com/d2/d2parser"
	d2color "oss.terrastruct.com/d2/lib/color"
)

type DocumentColor struct {
	Range Range `json:"range"`
	Color Color `json:"color"`
}

type Color struct {
	Red   float64 `json:"red"`
	Green float64 `json:"green"`
	Blue  float64 `json:"blue"`
	Alpha float64 `json:"alpha"`
}

type ColorPresentation struct {
	Label    string   `json:"label"`
	TextEdit TextEdit `json:"textEdit"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

func DocumentColors(path, text string) ([]DocumentColor, error) {
	ast, err := d2parser.Parse(path, strings.NewReader(text), &d2parser.ParseOptions{
		UTF16Pos: true,
	})
	if err != nil {
		var pe *d2parser.ParseError
		if !errors.As(err, &pe) {
			return nil, err
		}
	}
	if ast == nil {
		return nil, nil
	}

	var colors []DocumentColor
	collectDocumentColors(ast, &colors)
	return colors, nil
}

func collectDocumentColors(m *d2ast.Map, colors *[]DocumentColor) {
	for _, node := range m.Nodes {
		if node.MapKey == nil {
			continue
		}

		if color, ok := documentColorForKey(node.MapKey); ok {
			*colors = append(*colors, color)
		}
		if node.MapKey.Value.Map != nil {
			collectDocumentColors(node.MapKey.Value.Map, colors)
		}
	}
}

func documentColorForKey(mk *d2ast.Key) (DocumentColor, bool) {
	if !isColorKey(lastKeyPart(mk)) {
		return DocumentColor{}, false
	}

	scalar, ok := mk.Value.Unbox().(d2ast.Scalar)
	if !ok {
		return DocumentColor{}, false
	}

	value := scalar.ScalarString()
	if !d2color.ValidColor(value) || d2color.IsGradient(value) || d2color.IsThemeColor(value) {
		return DocumentColor{}, false
	}

	parsed, err := csscolorparser.Parse(value)
	if err != nil {
		return DocumentColor{}, false
	}

	return DocumentColor{
		Range: fromD2Range(scalar.GetRange()),
		Color: Color{
			Red:   parsed.R,
			Green: parsed.G,
			Blue:  parsed.B,
			Alpha: parsed.A,
		},
	}, true
}

func isColorKey(key string) bool {
	switch strings.ToLower(key) {
	case "fill", "stroke", "font-color":
		return true
	default:
		return false
	}
}

func ColorPresentations(color Color, r Range) []ColorPresentation {
	label := colorHex(color)
	return []ColorPresentation{{
		Label: label,
		TextEdit: TextEdit{
			Range:   r,
			NewText: label,
		},
	}}
}

func colorHex(color Color) string {
	red := colorComponent(color.Red)
	green := colorComponent(color.Green)
	blue := colorComponent(color.Blue)
	alpha := colorComponent(color.Alpha)
	if alpha < 255 {
		return fmt.Sprintf("#%02x%02x%02x%02x", red, green, blue, alpha)
	}
	return fmt.Sprintf("#%02x%02x%02x", red, green, blue)
}

func colorComponent(value float64) int {
	value = math.Max(0, math.Min(1, value))
	return int(math.Round(value * 255))
}
