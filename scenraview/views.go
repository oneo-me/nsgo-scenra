// Package scenraview gives NSGo apps support for Scenra files: an Editor
// that opens, edits, saves and undoes them, with an assistant that draws
// on them, and views of it that an app shows as Scenra's Window or lays
// out itself. The app supplies the model provider and keeps its own
// menus, settings and window.
package scenraview

import (
	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// The views show an Editor. An app composes them, or shows its Window;
// each follows the document the editor shows.

// TabBar is a tab for each open document, and a button that makes
// another. As the window's TitleBar, it drags the window and keeps clear
// of the window controls.
type TabBar struct {
	Editor   *Editor
	TitleBar bool
}

// Render implements core.Node.
func (t TabBar) Render(cx *core.Scope) core.Node { return titleBar(cx, t.Editor, t.TitleBar) }

// Sidebar lists the screens of the document shown, each with its design
// and flows, and opens the one chosen.
type Sidebar struct{ Editor *Editor }

// Render implements core.Node.
func (s Sidebar) Render(cx *core.Scope) core.Node { return sidebar(cx, s.Editor) }

// Workspace is what the Sidebar opened: a screen's design, on which the
// user comments, or a flow chart; or, while no file is open, a way to
// make or open one.
type Workspace struct{ Editor *Editor }

// Render implements core.Node.
func (w Workspace) Render(cx *core.Scope) core.Node { return workspacePane(cx, w.Editor) }

// Assistant is the conversation about the document shown, which draws
// on it. As the window's TitleBar, its header drags the window and keeps
// clear of the window controls.
type Assistant struct {
	Editor   *Editor
	TitleBar bool
}

// Render implements core.Node.
func (a Assistant) Render(cx *core.Scope) core.Node { return assistantPane(cx, a.Editor.a, a.TitleBar) }

// Window is Scenra's layout: the TabBar over the Sidebar and the
// Workspace, beside the Assistant running the window's height, whose
// width the user drags. With TitleBar, its top is the window's title bar.
type Window struct {
	Editor   *Editor
	TitleBar bool
}

// Render implements core.Node.
func (w Window) Render(cx *core.Scope) core.Node {
	tr := locales.For(cx)
	e := w.Editor
	chatWidth := core.NewSignal(cx, core.Points(400))
	return ui.ResizablePanelGroup{Style: ui.FillRest, HandleLabel: tr.Assistant(), Panels: []base.ResizablePanel{
		{MinSize: 640, Children: []core.Node{core.View{Style: ui.FillRest, Children: []core.Node{
			TabBar{Editor: e, TitleBar: w.TitleBar},
			core.View{Style: ui.FillRest.Merge(core.Style{Direction: core.Row}), Children: []core.Node{Sidebar{e}, Workspace{e}}},
		}}}},
		{Size: chatWidth.Get, OnResize: chatWidth.Set, MinSize: 320, MaxSize: 720, Children: []core.Node{Assistant{Editor: e, TitleBar: w.TitleBar}}},
	}}
}
