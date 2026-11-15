package scenraview

import (
	"path"
	"path/filepath"
	"slices"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// HeaderHeight is the height of the Header and of the Assistant's
// header; as the window's title bar, the system centers its window
// controls in it.
const HeaderHeight core.Points = 40

// chrome is the inset of a header that is the window's title bar, from
// the window controls, and whether its background drags the window; none
// for another.
func chrome(cx *core.Scope, titleBar bool) (core.Edges, core.WindowDrag) {
	if !titleBar {
		return core.Edges{}, core.WindowDragInherit
	}
	return desktop.TitleBar(cx), core.WindowDragOn
}

// header is the bar over the document shown: the open folder's name and
// the document's title, a dot while it has unsaved changes, and, when a
// design and its flow are both in the folder, a switch between them.
func header(cx *core.Scope, e *Editor, isTitleBar bool) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	// pair is the shown file's design and flow, relative to the folder,
	// when both are in it.
	pair := core.NewMemoFunc(cx, func() [2]string {
		d := e.cur()
		if d == e.blank {
			return [2]string{}
		}
		design, flow := pairOf(e.rel(d.path))
		files := e.files.Get()
		has := func(p string) bool { return slices.ContainsFunc(files, func(f entry) bool { return f.path == p }) }
		if !has(design) || !has(flow) {
			return [2]string{}
		}
		return [2]string{design, flow}
	}, func(a, b [2]string) bool { return a == b })
	showsFlow := func() bool { _, flow := stem(path.Base(e.cur().path)); return flow }
	toggle := ui.Tabs[bool]{Size: ui.SizeSmall, Label: tr.DesignOrFlow(), TestID: "pair",
		Options: []ui.Option[bool]{{Value: false, Label: tr.Design(), Icon: lucide.AppWindow, TestID: "pair-design"},
			{Value: true, Label: tr.Flow(), Icon: lucide.Workflow, TestID: "pair-flow"}},
		Value: showsFlow,
		OnChange: func(flow bool) {
			to := pair.GetUntracked()[0]
			if flow {
				to = pair.GetUntracked()[1]
			}
			e.open(filepath.Join(e.folder.GetUntracked(), filepath.FromSlash(to)))
		}}
	_, drag := chrome(cx, isTitleBar)
	return core.View{TestID: "header", Desktop: core.DesktopViewProps{WindowDrag: drag},
		StyleFunc: func() core.Style {
			bar, _ := chrome(cx, isTitleBar)
			return core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6, FlexShrink: 0,
				Height: core.Pt(max(HeaderHeight, bar.Top)), Padding: core.Edges{Left: bar.Left + 14, Right: 8},
				Background: ui.Chrome(cx), Borders: core.Edges{Bottom: 1}, BorderColors: core.EdgeColors{Bottom: colors.Line()}}
		},
		Children: []core.Node{
			core.Show{When: func() bool { return e.folder.Get() != "" }, Then: func(*core.Scope) core.Node {
				return core.Text{ContentFunc: func() string { return filepath.Base(e.folder.Get()) }, MaxLines: 1, TestID: "folder-name",
					Style:     ui.TypeBody.Merge(core.Style{FlexShrink: 1, MinWidth: core.Pt(0)}),
					StyleFunc: ui.Colored(colors.Muted)}
			}},
			core.Show{When: func() bool { return e.folder.Get() != "" && !e.empty() }, Then: func(*core.Scope) core.Node {
				return core.Text{Content: "/", Style: ui.TypeBody.Merge(core.Style{FlexShrink: 0}), StyleFunc: ui.Colored(colors.Muted)}
			}},
			core.Text{ContentFunc: e.Title, MaxLines: 1, TestID: "file-name",
				Style:     ui.TypeBody.Merge(core.Style{FontWeight: ui.Medium, FlexShrink: 1, MinWidth: core.Pt(0)}),
				StyleFunc: ui.Colored(colors.Text)},
			core.Show{When: func() bool { return e.cur().dirty.Get() }, Then: func(*core.Scope) core.Node {
				return core.View{Style: core.Style{Width: core.Pt(7), Height: core.Pt(7), CornerRadius: 3.5, FlexShrink: 0},
					StyleFunc: ui.Filled(colors.Muted), Label: tr.Edited(), TestID: "file-edited"}
			}},
			core.View{Style: core.Style{FlexGrow: 1}},
			core.Show{When: func() bool { return pair.Get()[0] != "" }, Then: func(*core.Scope) core.Node { return toggle }},
		}}
}
