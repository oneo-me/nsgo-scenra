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
func (s Sidebar) Render(cx *core.Scope) core.Node { return sidebar(cx, s.Editor) }

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

// Window is Scenra's layout: the Header over the Sidebar and the
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
			Header{Editor: e, TitleBar: w.TitleBar},
			core.View{Style: ui.FillRest.Merge(core.Style{Direction: core.Row}), Children: []core.Node{Sidebar{e}, Workspace{e}}},
		}}}},
		{Size: chatWidth.Get, OnResize: chatWidth.Set, MinSize: 320, MaxSize: 720, Children: []core.Node{Assistant{Editor: e, TitleBar: w.TitleBar}}},
	}}
}
