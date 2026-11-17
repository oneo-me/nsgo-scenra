package scenraview

import (
	"path/filepath"
	"slices"

	"github.com/nsgo-dev/nsgo/core"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// coverInset is the least room a Cover leaves around its design.
const coverInset = 12

// Cover draws a folder's project small, as a picture telling it from
// others, such as among the folders opened lately: the first design of
// File, when it is a file of designs in the project, or else of the
// project's first file of designs, scaled down to fit Style's box and
// centered in it, as Still draws it. It draws nothing while the folder is
// read, or when the project has no design.
type Cover struct {
	Folder, File string
	TestID       string
	Style        core.Style
}

// Render implements core.Node.
func (c Cover) Render(cx *core.Scope) core.Node {
	type project struct {
		file  flowfile.File
		parts components
		pics  assets
	}
	read := core.NewSignalFunc(cx, project{}, func(a, b project) bool { return false })
	go func() {
		root := projectDir(c.Folder)
		entries, parts, pics, _, _ := scan(root, nil)
		if f, ok := coverFile(root, c.File, entries); ok {
			cx.Post(func() { read.Set(project{f, parts, pics}) })
		}
	}()
	same := func(a, b core.Size) bool { return a == b }
	box := core.NewSignalFunc(cx, core.Size{}, same)
	size := core.NewSignalFunc(cx, core.Size{}, same)
	// The design shows once both sizes are known, at the scale fitting
	// the one in the other.
	scale := func() float32 {
		b, s := box.Get(), size.Get()
		if b.Width <= 0 || s.Width <= 0 || s.Height <= 0 {
			return 0
		}
		return float32(min(1, (b.Width-2*coverInset)/s.Width, (b.Height-2*coverInset)/s.Height))
	}
	drawing := core.Dynamic(func(cx *core.Scope) core.Node {
		p := read.Get()
		if len(p.file.Screens) == 0 {
			return nil
		}
		sel := stillSelection(cx, p.file.Screens[:1], p.parts, p.pics)
		sel.bare = func() bool { return true }
		return core.View{OnLayout: func(b core.Bounds) { size.Set(b.Size) },
			StyleFunc: func() core.Style {
				k, b, s := scale(), box.Get(), size.Get()
				st := core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), PointerEvents: core.PointerEventsNone,
					TransformOrigin: core.ObjectPosition{X: core.PositionAxis{Offset: core.Pt(0)}, Y: core.PositionAxis{Offset: core.Pt(0)}}}
				if k == 0 {
					st.Visibility = core.VisibilityHidden
					return st
				}
				st.Left, st.Top = core.Pt((b.Width-s.Width*core.Points(k))/2), core.Pt((b.Height-s.Height*core.Points(k))/2)
				st.Transform = core.Transform{Scale: k}
				return st
			},
			Children: []core.Node{screenView(cx, p.file, 0, sel)}}
	})
	return core.View{TestID: c.TestID, OnLayout: func(b core.Bounds) { box.Set(b.Size) },
		Style: core.Style{Overflow: core.Hidden}.Merge(c.Style), Children: []core.Node{drawing}}
}

// coverFile is the file a Cover of the project at root draws: the file
// at path when it holds designs, or else the first file of designs at
// the project's top, in the views folder or anywhere, as entries list
// them.
func coverFile(root, path string, entries []entry) (flowfile.File, bool) {
	designs := func(f flowfile.File, err error) bool {
		return err == nil && f.Kind == flowfile.Design && len(f.Screens) > 0
	}
	if path != "" {
		if rel, err := filepath.Rel(root, path); err == nil && filepath.IsLocal(rel) {
			if f, err := readFile(path); designs(f, err) {
				return f, true
			}
		}
	}
	rank := func(en entry) int {
		switch _, named := designNameOf(en.path); {
		case named && !isViewFile(en.path):
			return 0
		case named:
			return 1
		}
		return 2
	}
	entries = slices.DeleteFunc(slices.Clone(entries), func(en entry) bool { return en.kind != flowfile.Design })
	slices.SortStableFunc(entries, func(a, b entry) int { return rank(a) - rank(b) })
	for _, en := range entries {
		if f, err := readFile(filepath.Join(root, filepath.FromSlash(en.path))); designs(f, err) {
			return f, true
		}
	}
	return flowfile.File{}, false
}
