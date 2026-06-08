package d2features

import (
	"context"

	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2renderers/d2svg"
	"oss.terrastruct.com/d2/lib/log"
	"oss.terrastruct.com/d2/lib/memfs"
	"oss.terrastruct.com/d2/lib/textmeasure"
)

func RenderSVG(path, text string, files map[string]string) (string, error) {
	ruler, err := textmeasure.NewRuler()
	if err != nil {
		return "", err
	}

	compileOpts := &d2lib.CompileOptions{
		UTF16Pos:  true,
		InputPath: path,
		Ruler:     ruler,
		LayoutResolver: func(engine string) (d2graph.LayoutGraph, error) {
			return d2dagrelayout.DefaultLayout, nil
		},
	}
	if len(files) > 0 {
		fs, err := memfs.New(files)
		if err != nil {
			return "", err
		}
		compileOpts.FS = fs
	}

	pad := int64(5)
	renderOpts := &d2svg.RenderOpts{
		Pad: &pad,
	}
	ctx := log.WithDefault(context.Background())
	diagram, _, err := d2lib.Compile(ctx, text, compileOpts, renderOpts)
	if err != nil {
		return "", err
	}

	out, err := d2svg.Render(diagram, renderOpts)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
