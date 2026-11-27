// Package scenraview gives NSGo apps support for Scenra files: an Editor
// of a folder of them that opens, edits, saves and undoes them, with an
// assistant that draws in the folder, and views of it that an app shows
// as Scenra's Window or lays out itself. The app supplies the agents and
// keeps its own menus, settings and window.
package scenraview

import (
	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// The views show an Editor. An app composes them, or shows its Window;
// each follows the document the editor shows.

// Header names the document shown and switches between a design and its
// flow. As the window's TitleBar, it drags the window and keeps clear of
// the window controls.
type Header struct {
	Editor   *Editor
	TitleBar bool
}

// Render implements core.Node.
func (h Header) Render(cx *core.Scope) core.Node { return header(cx, h.Editor, h.TitleBar) }

// Sidebar is the open folder's Scenra files as a tree, which shows the
// one chosen and makes new ones.
type Sidebar struct{ Editor *Editor }

// Render implements core.Node.
func (s Sidebar) Render(cx *core.Scope) core.Node { return sidebar(cx, s.Editor, false) }

// Workspace is the file the Sidebar opened: a design, on which the user
// comments, or a flow chart; or, while none is, a way to open a folder or
// make a file.
type Workspace struct{ Editor *Editor }

// Render implements core.Node.
func (w Workspace) Render(cx *core.Scope) core.Node { return workspacePane(cx, w.Editor) }

// Assistant is the conversation about the open folder, which draws in
// it. As the window's TitleBar, its header drags the window and keeps
// clear of the window controls.
type Assistant struct {
	Editor   *Editor
	TitleBar bool
}

// Render implements core.Node.
func (a Assistant) Render(cx *core.Scope) core.Node { return assistantPane(cx, a.Editor.a, a.TitleBar) }

// Window is Scenra's layout: the Workspace filling it, under the Header
// over the Sidebar and beside the Assistant running the window's height,
// whose width the user drags; the Sidebar and the Assistant float over
// the Workspace as cards, which its canvases draw under but center clear
// of. With TitleBar, its top is the window's title bar. NoHeader leaves
// the Header out, for an app naming the document elsewhere, such as in
// its tabs. While the editor's PanelsHidden, the Header and Workspace
// fill it; the panels stay mounted, keeping the tree's and the
// conversation's state.
type Window struct {
	Editor   *Editor
	TitleBar bool
	NoHeader bool
}

// floatGap is how far a floating panel keeps from the window's edges and
// the workspace.
const floatGap core.Points = 8

// Render implements core.Node.
func (w Window) Render(cx *core.Scope) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	e := w.Editor
	chatWidth := core.NewSignal(cx, core.Points(400))
	shown := func() bool { return !e.PanelsHidden() }
	// What the panels and the Header cover of the Workspace, which its
	// canvases lay out clear of.
	header, chat := core.NewSignal(cx, core.Points(0)), core.NewSignal(cx, core.Points(0))
	cx.Effect(func() {
		in := core.Edges{Top: header.Get()}
		if shown() {
			in.Left, in.Right = floatGap+sidebarWidth, chat.Get()
		}
		core.Untracked(cx, func() bool { e.insets.Set(in); return true })
	})
	cx.Cleanup(func() { e.insets.Set(core.Edges{}) })
	fill := core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0)}
	// The panels' group lets the pointer through to the Workspace but
	// where a panel or its handle is.
	through := core.Style{PointerEvents: core.PointerEventsBoxNone}
	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{
		core.View{TestID: "workspace", Style: fill, Children: []core.Node{Workspace{e}}},
		ui.ResizablePanelGroup{Style: fill.Merge(through), HandleLabel: tr.Assistant(),
			// The cards' gap is the handle, which draws no line.
			HandleStyle: core.Style{Background: core.RGBA(0xffffff, 0)},
			Panels: []base.ResizablePanel{
				{MinSize: 640, Style: through, Children: []core.Node{core.View{Style: ui.FillRest.Merge(through), Children: []core.Node{
					core.Show{When: func() bool { return !w.NoHeader }, Then: func(*core.Scope) core.Node {
						return core.View{OnLayout: func(b core.Bounds) { header.Set(b.Size.Height) }, Children: []core.Node{Header{Editor: e, TitleBar: w.TitleBar}}}
					}},
					core.View{Style: ui.FillRest.Merge(through).Merge(core.Style{Direction: core.Row}), Children: []core.Node{
						core.View{StyleFunc: func() core.Style {
							if shown() {
								return core.Style{Direction: core.Row, FlexShrink: 0, Padding: core.Edges{Left: floatGap, Top: floatGap, Bottom: floatGap}}
							}
							return core.Style{Display: core.DisplayNone}
						}, Children: []core.Node{floatingCard(cx, colors.Surface, sidebar(cx, e, true))}},
					}},
				}}}},
				{Size: chatWidth.Get, Visible: shown, OnResize: chatWidth.Set, MinSize: 320, MaxSize: 720,
					OnLayout: func(b core.Bounds) { chat.Set(b.Size.Width) },
					Style:    core.Style{Padding: core.Edges{Top: floatGap, Right: floatGap, Bottom: floatGap}},
					Children: []core.Node{floatingCard(cx, colors.Background, Assistant{Editor: e, TitleBar: w.TitleBar})}},
			}},
	}}
}

// floatingCard is a panel floating over the workspace: rounded, edged and
// shadowed, on fill. The shadow is the card's and the clip its content's,
// so the shadow falls outside.
func floatingCard(cx *core.Scope, fill func() core.Color, panel core.Node) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{Style: ui.FillRest.Merge(core.Style{CornerRadius: ui.RadiusCard, OutlineWidth: 1}),
		StyleFunc: func() core.Style {
			return core.Style{Background: fill(), OutlineColor: colors.Line(), Shadows: []core.BoxShadow{
				ui.Shadow(0, 1, 2, 0, ui.Black(0.05)),
				ui.Shadow(0, 4, 16, 0, ui.Black(0.08)),
			}}
		},
		Children: []core.Node{core.View{Style: ui.FillRest.Merge(core.Style{CornerRadius: ui.RadiusCard, Overflow: core.Hidden}), Children: []core.Node{panel}}},
	}
}
