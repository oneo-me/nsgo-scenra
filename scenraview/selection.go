package scenraview

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// location is what the user selected in a file: a node of a flow, or an
// element of a design, by their keys, "" for none.
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

// Which notes get_view lists: all of the folder's, those of the file
// shown and the components it uses, or none, as they change seldom and
// an agent may have read them.
const (
	NotesAll  = "all"
	NotesOpen = "open"
	NotesNone = "none"
)

// view tells an agent what the user sees of the folder dir and the
// problems of its files: the file shown and what is selected in it, the
// components designs can use, the user's comments, the notes of the
// files notes says, the files that do not parse, the components missing
// or used in a loop, the pictures missing and the classes that draw
// nothing.
func view(dir string, files []fileView, parts components, pics assets, notes string) string {
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
		f := shown.file
		fmt.Fprintf(&b, "The user has flow %s open%s.\n", shown.path, holding(f))
		if flow, id, ok := splitNode(shown.shown.node); ok && flow < len(f.Flows) {
			fmt.Fprintf(&b, "The user has selected node %q of %s.\n", id, which(f, f.Flows[flow].ID))
		}
	default:
		f, kind := shown.file, shown.file.Kind
		fmt.Fprintf(&b, "The user has %s %s open%s.\n", kind, shown.path, holding(f))
		if el, ok := elementOf(f, shown.shown.element); ok {
			screen, path, _ := splitKey(shown.shown.element)
			fmt.Fprintf(&b, "The user has selected element %s of %s:\n\n```\n%s```\n", path, which(f, f.Screens[screen].ID),
				flowfile.FormatUI([]flowfile.Element{el}))
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
	// remark tells of a comment or note.
	remark := func(f fileView, c flowfile.Comment) string {
		if f.file.Several() {
			return fmt.Sprintf("%s, design %s, element %s: %q", f.path, f.file.Screens[c.Screen].ID, c.Path, c.Text)
		}
		return fmt.Sprintf("%s, element %s: %q", f.path, c.Path, c.Text)
	}
	section("The user's open comments:", func(f fileView) []string {
		var out []string
		for _, c := range f.file.Comments() {
			out = append(out, remark(f, c))
		}
		return out
	})
	noted := notedFiles(files, shown, parts, notes)
	section("The notes on elements, which say what the designs cannot show, such as what an element does or when it changes; keep them true:", func(f fileView) []string {
		if !noted(f) {
			return nil
		}
		var out []string
		for _, n := range f.file.Notes() {
			out = append(out, remark(f, n))
		}
		for _, n := range f.file.UseNotes() {
			out = append(out, remark(f, flowfile.Comment{Screen: n.Screen, Path: n.Path + ", its " + n.Name, Text: n.Text}))
		}
		return out
	})
	// opens are what taps open: a file of designs by its name, and each
	// of its designs with an ID by the name, # and the ID.
	var opens []string
	for _, f := range files {
		if f.file.Kind != flowfile.Chart {
			name := strings.TrimSuffix(f.path, ".scenra")
			opens = append(opens, name)
			for _, s := range f.file.Screens {
				if s.ID != "" {
					opens = append(opens, name+"#"+s.ID)
				}
			}
		}
	}
	docs := designs{}
	for _, f := range files {
		if f.file.Kind == flowfile.Design {
			docs[strings.TrimSuffix(f.path, ".scenra")] = f.file
		}
	}
	section("Taps that lead nowhere:", func(f fileView) []string {
		var out []string
		for _, p := range tapProblems(f.file, strings.TrimSuffix(f.path, ".scenra"), opens, docs, parts) {
			out = append(out, f.path+": "+p)
		}
		return out
	})
	section(fmt.Sprintf("Files in folders, which the editor does not show; keep designs at the top of the folder, flows in %s/ and components in %s/, and group designs and components with their names, such as name=\"Orders/Detail\":", flowsDir, flowfile.ComponentsDir), func(f fileView) []string {
		if inLayout(f.path) {
			return nil
		}
		return []string{f.path}
	})
	section(fmt.Sprintf("Flows outside %s/, and designs in it, where the editor does not pair them; keep a design's flow in %s/ under the design's file name:", flowsDir, flowsDir), func(f fileView) []string {
		if isFlowFile(f.path) == (f.file.Kind == flowfile.Chart) || f.file.Kind == "" || !inLayout(f.path) {
			return nil
		}
		return []string{f.path}
	})
	section("Names of groups in groups, which the editor shows ungrouped, as groups do not nest; keep one / in a name:", func(f fileView) []string {
		if name := f.file.Title(); nestedGroups(name) && !isFlowFile(f.path) {
			return []string{fmt.Sprintf("%s: name=%q", f.path, name)}
		}
		return nil
	})
	section("Files that do not parse, which the editor shows as they last parsed, if ever:", func(f fileView) []string {
		if f.broken == "" {
			return nil
		}
		return []string{f.path + ": " + f.broken}
	})
	section("Components that are missing or used in a loop, and what uses change or fill in them that is not there:", func(f fileView) []string {
		self, _ := flowfile.ComponentName(f.path)
		var out []string
		for _, s := range f.file.Screens {
			for _, p := range componentProblems(s, self, parts) {
				out = append(out, f.path+": "+p)
			}
		}
		if f.file.Kind == flowfile.Component && self == "" && inLayout(f.path) {
			out = append(out, fmt.Sprintf("%s: a component outside %s/, which nothing can use", f.path, flowfile.ComponentsDir))
		}
		return out
	})
	section("Outlets that in= cannot fill, and outlets of one name:", func(f fileView) []string {
		var out []string
		for _, p := range outletProblems(f.file, strings.TrimSuffix(f.path, ".scenra"), docs) {
			out = append(out, f.path+": "+p)
		}
		return out
	})
	section("Pictures that images cannot show:", func(f fileView) []string {
		var out []string
		for _, s := range f.file.Screens {
			for _, p := range imageProblems(s.UI, pics) {
				out = append(out, f.path+": "+p)
			}
		}
		return out
	})
	section("Classes that draw nothing:", func(f fileView) []string {
		var classes []string
		for _, s := range f.file.Screens {
			for _, c := range unknownClasses(s) {
				if !slices.Contains(classes, c) {
					classes = append(classes, c)
				}
			}
		}
		if len(classes) > 0 {
			return []string{f.path + ": " + strings.Join(classes, ", ")}
		}
		return nil
	})
	section("Suggestions, which change nothing drawn:", func(f fileView) []string {
		self, _ := flowfile.ComponentName(f.path)
		var out []string
		for _, s := range f.file.Screens {
			walk(s.UI, func(el flowfile.Element) {
				if part, ok := parts[el.Text]; ok && el.Kind == "use" && el.Text != self {
					for _, h := range flowfile.VariantHints(el, part) {
						out = append(out, f.path+": "+h)
					}
				}
			})
		}
		return out
	})
	return b.String()
}

// notedFiles says which of files get_view lists the notes of, as notes
// says, shown being the file shown, if any: those of the components it
// uses and those they use count as its own.
func notedFiles(files []fileView, shown *fileView, parts components, notes string) func(fileView) bool {
	switch {
	case notes == NotesNone:
		return func(fileView) bool { return false }
	case notes != NotesOpen:
		return func(fileView) bool { return true }
	case shown == nil:
		return func(fileView) bool { return false }
	}
	paths := []string{shown.path}
	var visit func(els []flowfile.Element)
	visit = func(els []flowfile.Element) {
		for _, name := range flowfile.Uses(els) {
			path := flowfile.ComponentPath(name)
			if part, ok := parts[name]; ok && !slices.Contains(paths, path) {
				paths = append(paths, path)
				visit(part.UI)
			}
		}
	}
	for _, s := range shown.file.Screens {
		visit(s.UI)
	}
	return func(f fileView) bool { return slices.Contains(paths, f.path) }
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

// holding lists the IDs of the designs or flows of a file of several.
func holding(f flowfile.File) string {
	if !f.Several() {
		return ""
	}
	var ids []string
	for _, s := range f.Screens {
		ids = append(ids, s.ID)
	}
	for _, d := range f.Flows {
		ids = append(ids, d.ID)
	}
	return fmt.Sprintf(", which holds the %ss %s", f.Kind, strings.Join(ids, ", "))
}

// which names the design or flow with id of f: by its ID in a file of
// several.
func which(f flowfile.File, id string) string {
	if f.Several() {
		return fmt.Sprintf("%s %s", f.Kind, id)
	}
	return "the " + string(f.Kind)
}

// tapProblems are the taps of the elements of f, named self, and those
// its uses give the elements of their components, that lead nowhere:
// those opening or closing none of designs, and in a design those
// closing a design it does not show in, by the in= of the designs of ds,
// or showing a panel neither it nor the components it uses has. A
// component's tabs may show the panels of the designs using it.
func tapProblems(f flowfile.File, self string, designs []string, ds designs, parts components) []string {
	var out []string
	for i, s := range f.Screens {
		// in are the designs this one shows in, itself first.
		in := []place{{self, i}}
		if f.Kind == flowfile.Design {
			hops, _ := ds.hosts(place{self, i})
			for _, h := range hops {
				in = append(in, h.place)
			}
		}
		panels := map[string]bool{}
		drawn(s.UI, parts, func(el flowfile.Element) {
			if name := el.Attr("panel"); name != "" {
				panels[name] = true
			}
		})
		check := func(key string, t flowfile.Tap) {
			file, id := t.Opens()
			if file == "" {
				file = self
			}
			to := file
			if id != "" {
				to += "#" + id
			}
			names := t.Action == flowfile.TapOpen || t.Action == flowfile.TapClose && t.Target != "" && t.Target != flowfile.Window
			switch {
			case names && !slices.Contains(designs, file):
				out = append(out, fmt.Sprintf("%s %q: there is no %s", key, t.String(), flowfile.DesignPath(file)))
			case names && !slices.Contains(designs, to):
				out = append(out, fmt.Sprintf("%s %q: %s has no design %s", key, t.String(), flowfile.DesignPath(file), id))
			case names && t.Action == flowfile.TapClose && f.Kind == flowfile.Design:
				at, _ := ds[file].Screen(id)
				if !slices.Contains(in, place{file, at}) {
					var titles []string
					for _, p := range in {
						titles = append(titles, ds.title(p))
					}
					out = append(out, fmt.Sprintf("%s %q: close names a design this one does not show in; it shows in %s, by their in=", key, t.String(), strings.Join(titles, ", ")))
				}
			case t.Action == flowfile.TapTab && f.Kind == flowfile.Design && !panels[t.Target]:
				out = append(out, fmt.Sprintf("%s %q: no element has panel=%s", key, t.String(), t.Target))
			}
		}
		walk(s.UI, func(el flowfile.Element) {
			if t, ok := el.Tap(); ok {
				check("tap", t)
			}
			for _, o := range el.Overrides() {
				if t, err := flowfile.ParseTap(o.Value); o.Attr == "tap" && err == nil {
					check(o.Key(), t)
				}
			}
		})
	}
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
// components they use, as the uses change them.
func drawn(els []flowfile.Element, parts components, fn func(flowfile.Element)) {
	var visit func(els []flowfile.Element, using []string)
	visit = func(els []flowfile.Element, using []string) {
		walk(els, func(el flowfile.Element) {
			fn(el)
			if part, ok := parts[el.Text]; ok && el.Kind == "use" && !slices.Contains(using, el.Text) {
				visit(flowfile.Instance(el, part), append(slices.Clone(using), el.Text))
			}
		})
	}
	visit(els, nil)
}
