package scenraview

import (
	"errors"
	"path/filepath"
	"slices"
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
		return deviceIcon(en.device)
	case flowfile.Component:
		return lucide.Component
	case flowfile.Chart:
		return lucide.Workflow
	}
	return lucide.FileX
}

// sidebar is the open folder's Scenra files as a tree, in which choosing
// a file shows it, with the lines Git says each changed, under a row
// showing its components together, which the tree leaves out; or, while no folder is open, a way
// to open one. It
// holds the dialog naming a new file, which the app's menu or the empty
// workspace asks for.
func sidebar(cx *core.Scope, e *Editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	// Components show in the showcase instead.
	tree := core.NewMemoFunc(cx, func() fileTree {
		return treeOf(slices.DeleteFunc(slices.Clone(e.files.Get()), func(f entry) bool { return isComponentFile(f.path) }))
	}, func(a, b fileTree) bool { return false })
	collapsed := core.NewSignalFunc(cx, []string(nil), slices.Equal)
	selection := ui.NewSelection(cx, false)
	// The tree selects the file shown, none while the components show, and
	// choosing a file shows it.
	cx.Effect(func() {
		d, showcase := e.cur(), e.showcase.Get()
		if d == e.blank && !showcase {
			return
		}
		key := e.rel(d.path)
		core.Untracked(cx, func() bool {
			switch {
			case showcase:
				selection.Clear()
				selection.SetCursor(nil)
			case selection.Cursor() != any(key):
				selection.Set(key)
			}
			return true
		})
	})
	cx.Effect(func() {
		key, _ := selection.Cursor().(string)
		cx.Untrack(func() {
			if key == "" || strings.HasSuffix(key, "/") || e.rel(e.shown().path) == key && !e.showcase.GetUntracked() {
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
		RowLabel: func(key any) string { return tree.Get().rowName(key.(string)) },
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
					core.Text{ContentFunc: func() string { return tree.Get().rowName(key()) }, MaxLines: 1,
						Style: ui.TypeBody.Merge(core.Style{FlexShrink: 1, MinWidth: core.Pt(0)}), StyleFunc: ui.Colored(colors.Text)},
					core.View{Style: core.Style{FlexGrow: 1}},
					changeCounts(cx, func() (change, bool) { c, ok := e.changes.Get()[key()]; return c, ok }),
				}}
		},
		Empty: core.View{Style: core.Style{Padding: core.Symmetric(4, 10)}, Children: []core.Node{
			core.Text{Content: tr.FilesEmpty(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)}}},
		Style: core.Style{FlexGrow: 1},
	}
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
					return core.View{Style: ui.FillRest.Merge(core.Style{Padding: core.Symmetric(0, 4)}), Children: []core.Node{
						core.Show{When: e.HasComponents, Then: func(cx *core.Scope) core.Node { return componentsRow(cx, e) }},
						files,
					}}
				}},
			newFileDialog(cx, e),
		}}
}

// componentsRow shows the open folder's components together, and is
// selected while they or one of them show.
func componentsRow(cx *core.Scope, e *Editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	count := func() int { return len(componentEntries(e.files.Get())) }
	return base.Button{TestID: "sidebar-components", Label: tr.Components(), OnPress: func(core.PressEvent) { e.ShowComponents() },
		Selected: func() bool {
			return e.showcase.Get() || e.cur() != e.blank && isComponentFile(e.rel(e.cur().path))
		},
		Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6, MinHeight: core.Pt(ui.RowHeight - 8),
			Padding: core.Symmetric(0, 10), Margin: core.Symmetric(0, 4), CornerRadius: ui.RadiusControl, FlexShrink: 0},
		StyleFunc: func() core.Style {
			return core.Style{Hover: ui.Hovered(cx), Selected: &core.Style{Background: ui.Alpha(colors.Text(), 0.1)}}
		},
		Children: []core.Node{
			ui.IconOf(lucide.LayoutGrid, 14, colors.Muted),
			core.Text{Content: tr.Components(), MaxLines: 1, Style: ui.TypeBody.Merge(core.Style{FlexShrink: 1, MinWidth: core.Pt(0)}),
				StyleFunc: ui.Colored(colors.Text)},
			core.View{Style: core.Style{FlexGrow: 1}},
			core.Text{ContentFunc: func() string { return strconv.Itoa(count()) }, Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
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
