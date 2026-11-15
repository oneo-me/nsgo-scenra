package scenraview

import (
	"fmt"
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
// user's comments, the files that do not parse and the classes that draw
// nothing.
func view(dir string, files []fileView) string {
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
	case shown.file.Kind == flowfile.Chart:
		fmt.Fprintf(&b, "The user has flow %s open.\n", shown.path)
		if n := shown.shown.node; n != "" {
			fmt.Fprintf(&b, "The user has selected node %q of the flow.\n", n)
		}
	default:
		fmt.Fprintf(&b, "The user has design %s open.\n", shown.path)
		if el, ok := flowfile.ElementAt(shown.file.UI, shown.shown.element); ok {
			fmt.Fprintf(&b, "The user has selected element %s of the design:\n\n```\n%s```\n", shown.shown.element, flowfile.FormatUI([]flowfile.Element{el}))
		}
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
	section("Files that do not parse, which the editor shows as they last parsed:", func(f fileView) []string {
		if f.broken == "" {
			return nil
		}
		return []string{f.path + ": " + f.broken}
	})
	section("Classes that draw nothing:", func(f fileView) []string {
		if classes := unknownClasses(f.file.UI); len(classes) > 0 {
			return []string{f.path + ": " + strings.Join(classes, ", ")}
		}
		return nil
	})
	return b.String()
}
