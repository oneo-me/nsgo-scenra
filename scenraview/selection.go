package scenraview

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// location is what the user selected in a file: a node of a flow, or the
// path of an element of a design, "" for none.
type location struct{ node, element string }

// fileView is what the server tells of a Scenra file of the open folder:
// its path relative to the folder, its content as the editor has it, why
// its latest version does not parse, and what the user selected in it
// while it shows.
type fileView struct {
	path   string
	file   flowfile.File
	broken string
	shown  *location
}

// view tells an agent what the user sees of the folder dir and the
// problems of its files: the file shown and what is selected in it, the
// components designs can use, the user's comments, the files that do not parse, the components missing
// or used in a loop, the pictures missing and the classes that draw
// nothing.
func view(dir string, files []fileView, parts components, pics assets) string {
	var b strings.Builder
	fmt.Fprintf(&b, "The open folder is %s; the paths below are relative to it.\n", dir)
	if len(files) == 0 {
		b.WriteString("It has no Scenra files yet.\n")
	}
	var shown *fileView
	for i := range files {
		if files[i].shown != nil {
			shown = &files[i]
		}
	}
	switch {
	case shown == nil:
		b.WriteString("The user has no file open.\n")
	case shown.file.Kind == "":
		fmt.Fprintf(&b, "The user has %s open, which has never parsed, so the editor shows its error instead.\n", shown.path)
	case shown.file.Kind == flowfile.Chart:
		fmt.Fprintf(&b, "The user has flow %s open.\n", shown.path)
		if n := shown.shown.node; n != "" {
			fmt.Fprintf(&b, "The user has selected node %q of the flow.\n", n)
		}
	default:
		kind := shown.file.Kind
		fmt.Fprintf(&b, "The user has %s %s open.\n", kind, shown.path)
		if el, ok := flowfile.ElementAt(shown.file.UI, shown.shown.element); ok {
			fmt.Fprintf(&b, "The user has selected element %s of the %s:\n\n```\n%s```\n", shown.shown.element, kind, flowfile.FormatUI([]flowfile.Element{el}))
		}
	}
	if len(parts) > 0 {
		names := slices.Sorted(maps.Keys(parts))
		fmt.Fprintf(&b, "Components, which designs and components draw with use <name>: %s.\n", strings.Join(names, ", "))
	}
	section := func(title string, line func(f fileView) []string) {
		var lines []string
		for _, f := range files {
			lines = append(lines, line(f)...)
		}
		if len(lines) > 0 {
			b.WriteString(title + "\n")
			for _, l := range lines {
				b.WriteString("- " + l + "\n")
			}
		}
	}
	section("The user's open comments:", func(f fileView) []string {
		var out []string
		for _, c := range f.file.Comments() {
			out = append(out, fmt.Sprintf("%s, element %s: %q", f.path, c.Path, c.Text))
		}
		return out
	})
	section("The notes on elements, which say what the designs cannot show, such as what an element does or when it changes; keep them true:", func(f fileView) []string {
		var out []string
		for _, n := range f.file.Notes() {
			out = append(out, fmt.Sprintf("%s, element %s: %q", f.path, n.Path, n.Text))
		}
		return out
	})
	var designs []string
	for _, f := range files {
		if f.file.Kind != flowfile.Chart {
			designs = append(designs, strings.TrimSuffix(f.path, ".scenra"))
		}
	}
	section("Taps that lead nowhere:", func(f fileView) []string {
		var out []string
		for _, p := range tapProblems(f.file, designs, parts) {
			out = append(out, f.path+": "+p)
		}
		return out
	})
	section("Files that do not parse, which the editor shows as they last parsed, if ever:", func(f fileView) []string {
		if f.broken == "" {
			return nil
		}
		return []string{f.path + ": " + f.broken}
	})
	section("Components that are missing or used in a loop, and what uses change in them that is not there:", func(f fileView) []string {
		self, _ := flowfile.ComponentName(f.path)
		var out []string
		for _, p := range componentProblems(f.file.UI, self, parts) {
			out = append(out, f.path+": "+p)
		}
		if f.file.Kind == flowfile.Component && self == "" {
			out = append(out, fmt.Sprintf("%s: a component outside %s/, which nothing can use", f.path, flowfile.ComponentsDir))
		}
		return out
	})
	section("Pictures that images cannot show:", func(f fileView) []string {
		var out []string
		for _, p := range imageProblems(f.file.UI, pics) {
			out = append(out, f.path+": "+p)
		}
		return out
	})
	section("Classes that draw nothing:", func(f fileView) []string {
		if classes := unknownClasses(f.file.UI); len(classes) > 0 {
			return []string{f.path + ": " + strings.Join(classes, ", ")}
		}
		return nil
	})
	return b.String()
}

// imageProblems are what is wrong with the pictures the images of els,
// and the uses changing theirs, name in the folder: those outside the
// assets folder or of a kind it does not show, and those missing.
func imageProblems(els []flowfile.Element, pics assets) []string {
	var out []string
	for _, src := range flowfile.Images(els) {
		p, ok := flowfile.AssetPath(src)
		switch {
		case !ok:
		case !flowfile.IsAsset(p):
			out = append(out, fmt.Sprintf("src %q: put the picture in %s/, as a file ending in %s, and name it there", src, flowfile.AssetsDir, strings.Join(flowfile.AssetExts, ", ")))
		case pics[p].data == nil:
			out = append(out, fmt.Sprintf("src %q: there is no %s", src, p))
		}
	}
	return out
}

// tapProblems are the taps of f's elements, and those its uses give the
// elements of their components, that lead nowhere: those
// opening none of designs, and in a design those showing a panel neither
// it nor the components it uses has. A component's tabs may show the
// panels of the designs using it.
func tapProblems(f flowfile.File, designs []string, parts components) []string {
	var out []string
	panels := map[string]bool{}
	drawn(f.UI, parts, func(el flowfile.Element) {
		if name := el.Attr("panel"); name != "" {
			panels[name] = true
		}
	})
	check := func(key string, t flowfile.Tap) {
		switch {
		case t.Action == flowfile.TapOpen && !slices.Contains(designs, t.Target):
			out = append(out, fmt.Sprintf("%s %q: there is no %s", key, t.String(), flowfile.DesignPath(t.Target)))
		case t.Action == flowfile.TapTab && f.Kind == flowfile.Design && !panels[t.Target]:
			out = append(out, fmt.Sprintf("%s %q: no element has panel=%s", key, t.String(), t.Target))
		}
	}
	walk(f.UI, func(el flowfile.Element) {
		if t, ok := el.Tap(); ok {
			check("tap", t)
		}
		for _, o := range el.Overrides() {
			if t, err := flowfile.ParseTap(o.Value); o.Attr == "tap" && err == nil {
				check(o.Key(), t)
			}
		}
	})
	return out
}

// walk runs fn on els and the elements they hold.
func walk(els []flowfile.Element, fn func(flowfile.Element)) {
	for _, el := range els {
		fn(el)
		walk(el.Children, fn)
	}
}

// drawn runs fn on els, the elements they hold and those of the
// components they use, each component once.
func drawn(els []flowfile.Element, parts components, fn func(flowfile.Element)) {
	seen := map[string]bool{}
	var visit func(els []flowfile.Element)
	visit = func(els []flowfile.Element) {
		walk(els, func(el flowfile.Element) {
			fn(el)
			if el.Kind == "use" && !seen[el.Text] {
				seen[el.Text] = true
				visit(parts[el.Text])
			}
		})
	}
	visit(els)
}
