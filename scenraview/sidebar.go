package scenraview

import (
	"errors"
	"path/filepath"
	"slices"
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
// flow's, or one of a file that does not parse.
func entryIcon(en entry) func() *core.SVG {
	switch en.kind {
	case flowfile.Design:
		return deviceIcon(en.device)
	case flowfile.Chart:
		return lucide.Workflow
	}
	return lucide.FileX
}

// sidebar is the open folder's name and its Scenra files as a tree, in
// which choosing a file shows it; or, while no folder is open, a way to
// open one. It holds the dialog naming a new file.
func sidebar(cx *core.Scope, e *Editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	tree := core.NewMemoFunc(cx, func() fileTree { return treeOf(e.files.Get()) }, func(a, b fileTree) bool { return false })
	collapsed := core.NewSignalFunc(cx, []string(nil), slices.Equal)
	selection := ui.NewSelection(cx, false)
	// The tree selects the file shown, and choosing a file shows it.
	cx.Effect(func() {
		d := e.cur()
		if d == e.blank {
			return
		}
		key := e.rel(d.path)
		core.Untracked(cx, func() bool {
			if selection.Cursor() != any(key) {
				selection.Set(key)
			}
			return true
		})
	})
	cx.Effect(func() {
		key, _ := selection.Cursor().(string)
		cx.Untrack(func() {
			if key == "" || strings.HasSuffix(key, "/") || e.rel(e.shown().path) == key {
				return
			}
			if _, ok := tree.GetUntracked().files[key]; ok {
				e.open(filepath.Join(e.folder.GetUntracked(), filepath.FromSlash(key)))
			}
		})
	})
	// New files go in the folder selected, or the one of the file
	// selected.
	e.into = ""
	cx.Effect(func() {
		key, _ := selection.Cursor().(string)
		e.into = folderOf(key)
	})
	keys := func(of string) []any {
		var out []any
		for _, k := range tree.Get().children[of] {
			out = append(out, k)
		}
		return out
	}
	files := ui.Tree{TestID: "files", Label: tr.Files(), ToggleLabel: tr.Expand(), Selection: selection,
		Roots:    func() []any { return keys("") },
		Children: func(key any) []any { return keys(key.(string)) },
		Expanded: func(key any) bool { return !slices.Contains(collapsed.Get(), key.(string)) },
		OnExpand: func(key any, open bool) {
			next := slices.DeleteFunc(slices.Clone(collapsed.GetUntracked()), func(k string) bool { return k == key })
			if !open {
				next = append(next, key.(string))
			}
			collapsed.Set(next)
		},
		RowLabel: func(key any) string { return keyName(key.(string)) },
		Row: func(cx *core.Scope, item ui.TreeItem) core.Node {
			key := func() string { return item.Key().(string) }
			icon := func() *core.SVG {
				if item.Branch() {
					if item.Expanded() {
						return lucide.FolderOpen()
					}
					return lucide.Folder()
				}
				return entryIcon(tree.Get().files[key()])()
			}
			return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6, FlexGrow: 1, MinWidth: core.Pt(0)},
				Children: []core.Node{
					ui.IconOf(icon, 14, colors.Muted),
					core.Text{ContentFunc: func() string { return keyName(key()) }, MaxLines: 1,
						Style: ui.TypeBody.Merge(core.Style{FlexShrink: 1, MinWidth: core.Pt(0)}), StyleFunc: ui.Colored(colors.Text)},
				}}
		},
		Empty: core.View{Style: core.Style{Padding: core.Symmetric(4, 10)}, Children: []core.Node{
			core.Text{Content: tr.FilesEmpty(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)}}},
		Style: core.Style{FlexGrow: 1},
	}
	header := core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4, Padding: core.Edges{Left: 12, Right: 6, Bottom: 4}},
		Children: []core.Node{
			core.Text{ContentFunc: func() string { return filepath.Base(e.folder.Get()) }, MaxLines: 1, TestID: "folder-name",
				Style: ui.TypeBody.Merge(core.Style{FontWeight: ui.Semibold, FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)}), StyleFunc: ui.Colored(colors.Text)},
			ui.Menu{Label: tr.NewFile(), Align: core.AlignEnd, TestID: "new-file",
				Trigger: ui.Button{Icon: lucide.Plus, Label: tr.NewFile(), Variant: ui.ButtonGhost, Side: 26},
				Children: []core.Node{
					ui.MenuItem{Text: tr.NewDesign(), Icon: lucide.AppWindow, OnSelect: e.NewDesign, TestID: "new-design"},
					ui.MenuItem{Text: tr.NewFlow(), Icon: lucide.Workflow, OnSelect: e.NewFlow, TestID: "new-flow"},
				}},
			ui.IconButton{Icon: lucide.FolderOpen, Label: tr.OpenFolder(), Size: 26, OnPress: e.OpenFolder, TestID: "open-folder"},
		}}
	noFolder := core.View{Style: core.Style{Padding: core.Symmetric(8, 12), Gap: 10}, Children: []core.Node{
		core.Text{Content: tr.NoFolder(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
		ui.Button{Text: tr.OpenFolder(), Icon: lucide.FolderOpen, Variant: ui.ButtonOutline, OnPress: e.OpenFolder, TestID: "sidebar-open-folder"},
	}}
	return core.View{Style: core.Style{Width: core.Pt(sidebarWidth), FlexShrink: 0, Padding: core.Symmetric(8, 0), Borders: core.Edges{Right: 1}},
		StyleFunc: func() core.Style {
			return core.Style{Background: colors.Surface(), BorderColors: core.EdgeColors{Right: colors.Line()}}
		},
		Children: []core.Node{
			core.Show{When: func() bool { return e.folder.Get() == "" },
				Then: func(*core.Scope) core.Node { return noFolder },
				Else: func(*core.Scope) core.Node {
					return core.View{Style: ui.FillRest, Children: []core.Node{header, core.View{Style: core.Style{FlexGrow: 1, Padding: core.Symmetric(0, 4)}, Children: []core.Node{files}}}}
				}},
			newFileDialog(cx, e),
		}}
}

// newFileDialog asks the name of the file of the kind e.naming says,
// which it makes in the folder e.into names and opens.
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
					if design, flow := stem(d.name()); !flow {
						name.Set(design)
					}
				}
				problem.Set("")
				return true
			})
		})
		create := func() {
			err := e.create(k, e.into, name.GetUntracked())
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
		if k == flowfile.Chart {
			title = tr.NewFlow()
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
// open a folder, or to make a file in the one open.
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
		core.Show{When: func() bool { return e.cur().doc.Get().Kind == flowfile.Design }, Then: func(cx *core.Scope) core.Node {
			return core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0)},
				Children: []core.Node{designPane(cx, e, e.a.ask())}}
		}},
	}}
	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{
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
	}}
}
