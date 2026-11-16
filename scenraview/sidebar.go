package scenraview

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

const sidebarWidth = 240

// deviceIcon is the icon of a design's device.
func deviceIcon(device string) func() *core.SVG {
	switch device {
	case "tablet":
		return lucide.Tablet
	case "desktop":
		return lucide.Monitor
	}
	return lucide.Smartphone
}

// entryIcon is the icon of a file of the tree: its design's device, a
// component's, a flow's, or one of a file that does not parse.
func entryIcon(en entry) func() *core.SVG {
	switch en.kind {
	case flowfile.Design:
		return deviceIcon(en.screen().Device)
	case flowfile.Component:
		return lucide.Component
	case flowfile.Chart:
		return lucide.Workflow
	}
	return lucide.FileX
}

// componentsItem is the sidebar's ID of the row showing the components
// together, which no file's path can be.
const componentsItem = "\x00components"

// sidebar is the open folder's flows in a group of their own pinned at
// its top, over a row showing its components together, which the list
// leaves out, and its designs in the groups their names give them; in
// which choosing a file shows it, with the lines Git says each changed;
// or, while no folder is open, a way to open one.
// It holds the dialog naming a new file, which the app's menu or the
// empty workspace asks for.
func sidebar(cx *core.Scope, e *Editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	groups := core.NewMemoFunc(cx, func() []fileGroup { return groupFiles(e.files.Get()) }, func(a, b []fileGroup) bool { return false })
	flows := core.NewMemoFunc(cx, func() []entry { return flowEntries(e.files.Get()) }, func(a, b []entry) bool { return false })
	row := func(cx *core.Scope, en entry, name string) core.Node {
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6, FlexGrow: 1, MinWidth: core.Pt(0)},
			Children: []core.Node{
				ui.IconOf(entryIcon(en), 14, colors.Muted),
				core.Text{Content: name, MaxLines: 1,
					Style: ui.TypeBody.Merge(core.Style{FlexShrink: 1, MinWidth: core.Pt(0)}), StyleFunc: ui.Colored(colors.Text)},
				core.View{Style: core.Style{FlexGrow: 1}},
				changeCounts(cx, func() (change, bool) { c, ok := e.changes.Get()[en.path]; return c, ok }),
			}}
	}
	components := ui.SidebarItem{ID: componentsItem, Text: tr.Components(), TestID: "sidebar-components",
		Content: core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6, FlexGrow: 1, MinWidth: core.Pt(0)},
			Children: []core.Node{
				ui.IconOf(lucide.LayoutGrid, 14, colors.Muted),
				core.Text{Content: tr.Components(), MaxLines: 1, Style: ui.TypeBody.Merge(core.Style{FlexShrink: 1, MinWidth: core.Pt(0)}),
					StyleFunc: ui.Colored(colors.Text)},
				core.View{Style: core.Style{FlexGrow: 1}},
				core.Text{ContentFunc: func() string { return strconv.Itoa(len(componentEntries(e.files.Get()))) },
					Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
			}}}
	sections := func() []ui.SidebarSection {
		gs := groups.Get()
		fs := flows.Get()
		flowSection := ui.SidebarSection{Title: tr.Flows(), Collapsible: true, TestID: "files-flows"}
		for _, en := range fs {
			flowSection.Items = append(flowSection.Items, ui.SidebarItem{ID: en.path, Text: en.title(), TestID: "file", Content: row(cx, en, en.title())})
		}
		top := ui.SidebarSection{TestID: "files-top"}
		if e.HasComponents() {
			top.Items = append(top.Items, components)
		}
		out := []ui.SidebarSection{top}
		for _, g := range gs {
			var items []ui.SidebarItem
			for _, en := range g.files {
				_, name := groupOf(en.title())
				items = append(items, ui.SidebarItem{ID: en.path, Text: name, TestID: "file", Content: row(cx, en, name)})
			}
			if g.name == "" {
				out[0].Items = append(out[0].Items, items...)
				continue
			}
			out = append(out, ui.SidebarSection{Title: g.name, Items: items, Collapsible: true, TestID: "group-" + g.name})
		}
		if len(out[0].Items) == 0 && len(gs) == 0 && len(fs) == 0 {
			out[0].Note = core.Text{Content: tr.FilesEmpty(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)}
		}
		if len(fs) > 0 {
			out = append([]ui.SidebarSection{flowSection}, out...)
		}
		return out
	}
	// The list marks the file shown, or the components while they or one
	// of them show, and choosing a file shows it.
	selected := func() string {
		d := e.cur()
		if e.showcase.Get() || d != e.blank && isComponentFile(e.rel(d.path)) {
			return componentsItem
		}
		if d == e.blank {
			return ""
		}
		return e.rel(d.path)
	}
	choose := func(id string) {
		if id == componentsItem {
			e.ShowComponents()
			return
		}
		e.open(filepath.Join(e.folder.GetUntracked(), filepath.FromSlash(id)))
	}
	files := ui.Sidebar{TestID: "files", SectionsFunc: sections, Selected: selected, OnSelect: choose}
	noFolder := core.View{Style: core.Style{Padding: core.Symmetric(8, 12), Gap: 10}, Children: []core.Node{
		core.Text{Content: tr.NoFolder(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
		ui.Button{Text: tr.OpenFolder(), Icon: lucide.FolderOpen, Variant: ui.ButtonOutline, OnPress: e.OpenFolder, TestID: "sidebar-open-folder"},
	}}
	return core.View{Style: core.Style{Width: core.Pt(sidebarWidth), FlexShrink: 0, Padding: core.Edges{Top: 8}, Borders: core.Edges{Right: 1}},
		StyleFunc: func() core.Style {
			return core.Style{Background: colors.Surface(), BorderColors: core.EdgeColors{Right: colors.Line()}}
		},
		Children: []core.Node{
			core.Show{When: func() bool { return e.folder.Get() == "" },
				Then: func(*core.Scope) core.Node { return noFolder },
				Else: func(*core.Scope) core.Node { return files }},
			newFileDialog(cx, e),
		}}
}

// changeCounts are the lines a file added and removed since the last
// commit, as Git tells them, nothing while it tells none.
func changeCounts(cx *core.Scope, of func() (change, bool)) core.Node {
	colors := ui.ColorsOf(cx)
	count := func(sign string, n func(change) int, color func() core.Color, testID string) core.Node {
		return core.Show{When: func() bool { c, ok := of(); return ok && n(c) > 0 }, Then: func(*core.Scope) core.Node {
			return core.Text{ContentFunc: func() string { c, _ := of(); return sign + strconv.Itoa(n(c)) }, TestID: testID,
				Style: ui.TypeCaption.Merge(core.Style{FlexShrink: 0, FontVariant: core.FontTabularNums}), StyleFunc: ui.Colored(color)}
		}}
	}
	return core.View{Style: core.Style{Direction: core.Row, Gap: 4, FlexShrink: 0}, Children: []core.Node{
		count("+", func(c change) int { return c.added }, colors.Up, "change-added"),
		count("−", func(c change) int { return c.removed }, colors.Down, "change-removed"),
	}}
}

// newFileDialog asks the name of the file of the kind e.naming says,
// which it makes and opens.
func newFileDialog(cx *core.Scope, e *Editor) core.Node {
	// The dialog is built for the kind last asked for, which it keeps
	// while it closes.
	kind := core.NewSignal(cx, flowfile.Design)
	cx.Effect(func() {
		if k := e.naming.Get(); k != "" {
			core.Untracked(cx, func() bool { kind.Set(k); return true })
		}
	})
	return core.Dynamic(func(cx *core.Scope) core.Node {
		k := kind.Get()
		tr := locales.For(cx)
		colors := ui.ColorsOf(cx)
		name := core.NewSignal(cx, "")
		problem := core.NewSignal(cx, "")
		// A new flow is named for the design shown, as its flow.
		cx.Effect(func() {
			e.naming.Get()
			core.Untracked(cx, func() bool {
				name.Set("")
				if d := e.shown(); k == flowfile.Chart && d.doc.GetUntracked().Kind == flowfile.Design {
					name.Set(keyName(d.path))
				}
				problem.Set("")
				return true
			})
		})
		create := func() {
			err := e.create(k, name.GetUntracked())
			switch {
			case errors.Is(err, errExists):
				problem.Set(tr.FileExists())
			case errors.Is(err, errName):
				problem.Set(tr.FileNameInvalid())
			case err != nil:
				problem.Set(tr.SaveFailed(err.Error()))
			default:
				e.naming.Set("")
			}
		}
		title := tr.NewDesign()
		switch k {
		case flowfile.Chart:
			title = tr.NewFlow()
		case flowfile.Component:
			title = tr.NewComponent()
		}
		return ui.Dialog{Open: func() bool { return e.naming.Get() != "" }, OnOpenChange: func(open bool, _ base.Reason) {
			if !open {
				e.naming.Set("")
			}
		}, TestID: "new-file-dialog", Title: title, Style: core.Style{Width: core.Pt(400)},
			Children: []core.Node{
				ui.Input{Value: name.Get, OnChange: func(v string) { name.Set(v); problem.Set("") }, OnSubmit: func(string) { create() },
					Placeholder: tr.FileName(), Label: tr.FileName(), AutoFocus: true, TestID: "new-file-name"},
				core.Show{When: func() bool { return problem.Get() != "" }, Then: func(*core.Scope) core.Node {
					return core.Text{ContentFunc: problem.Get, Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Danger), TestID: "new-file-problem"}
				}},
			},
			Footer: []core.Node{
				ui.Button{Text: tr.Cancel(), Variant: ui.ButtonOutline, OnPress: func() { e.naming.Set("") }, TestID: "new-file-cancel"},
				ui.Button{Text: tr.Create(), Variant: ui.ButtonPrimary, Disabled: func() bool { return strings.TrimSpace(name.Get()) == "" },
					OnPress: create, TestID: "new-file-create"},
			}}
	})
}

// workspacePane is the file shown: a design, or a flow chart; or a way to
// open a folder, or to make a file in the one open; or the folder's
// components together.
func workspacePane(cx *core.Scope, e *Editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	message := func(icon func() *core.SVG, text string, actions ...core.Node) core.Node {
		return core.View{Style: core.Style{FlexGrow: 1, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Padding: core.All(32), Gap: 12},
			Children: []core.Node{
				ui.Icon(icon, 32, colors.Muted),
				core.Text{Content: text, Style: core.Style{FontSize: 13, LineHeight: 19, TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(360)},
					StyleFunc: ui.Colored(colors.Muted)},
				core.View{Style: core.Style{Direction: core.Row, Gap: 8, Margin: core.Edges{Top: 4}}, Children: actions},
			}}
	}
	// The flow pane stays while documents change, as remounting a chart
	// trips NSGo; the design covers it while one shows.
	body := core.View{Style: ui.FillRest, Children: []core.Node{
		flowPane(cx, e),
		core.Show{When: func() bool { k := e.cur().doc.Get().Kind; return k == flowfile.Design || k == flowfile.Component }, Then: func(cx *core.Scope) core.Node {
			return core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0)},
				Children: []core.Node{designPane(cx, e, e.a.ask())}}
		}},
		core.Show{When: func() bool { return e.cur().unread() }, Then: func(cx *core.Scope) core.Node {
			return core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0)},
				StyleFunc: ui.Filled(colors.Background), Children: []core.Node{brokenPane(cx, e, e.a.ask())}}
		}},
	}}
	// The components cover the document while they show, which stays, as
	// its flow pane must.
	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{
		probeLayer(cx, e),
		core.Show{When: e.empty,
			Then: func(*core.Scope) core.Node {
				return core.Show{When: func() bool { return e.folder.Get() == "" },
					Then: func(*core.Scope) core.Node {
						return message(lucide.FolderOpen, tr.NoFolder(),
							ui.Button{Text: tr.OpenFolder(), Icon: lucide.FolderOpen, Variant: ui.ButtonPrimary, OnPress: e.OpenFolder, TestID: "open-folder-start"})
					},
					Else: func(*core.Scope) core.Node {
						return message(lucide.FileStack, tr.NoFileOpen(),
							ui.Button{Text: tr.NewDesign(), Icon: lucide.AppWindow, Variant: ui.ButtonOutline, OnPress: e.NewDesign, TestID: "start-new-design"},
							ui.Button{Text: tr.NewFlow(), Icon: lucide.Workflow, Variant: ui.ButtonOutline, OnPress: e.NewFlow, TestID: "start-new-flow"})
					}}
			},
			Else: func(*core.Scope) core.Node { return body }},
		core.Show{When: func() bool { return e.showcase.Get() && e.folder.Get() != "" }, Then: func(cx *core.Scope) core.Node {
			return core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0)},
				Children: []core.Node{showcasePane(cx, e)}}
		}},
	}}
}

// brokenPane tells why the file shown does not parse, as it never has,
// and with ask asks the assistant to fix it; the file shows once it
// parses.
func brokenPane(cx *core.Scope, e *Editor, ask func(prompt string)) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	problem := func() string { return e.cur().broken.Get() }
	return core.View{TestID: "file-broken", Style: core.Style{FlexGrow: 1, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Padding: core.All(32), Gap: 12},
		Children: []core.Node{
			ui.Icon(lucide.FileExclamationPoint, 32, colors.Danger),
			ui.Text(tr.FileBroken(), 15, ui.Semibold, colors.Text),
			core.Text{ContentFunc: problem, TestID: "file-problem", Style: ui.TypeCode.Merge(core.Style{TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(480)}),
				StyleFunc: ui.Colored(colors.Muted)},
			core.Show{When: func() bool { return ask != nil }, Then: func(*core.Scope) core.Node {
				return ui.Button{Text: tr.FixFile(), Icon: lucide.Sparkles, Variant: ui.ButtonPrimary, TestID: "fix-file",
					Disabled: e.Working,
					OnPress: func() {
						d := e.shown()
						ask(tr.FixFilePrompt(e.rel(d.path), d.broken.GetUntracked()))
					}}
			}},
		}}
}
