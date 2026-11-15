package scenraview

import (
	"path/filepath"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	agio "github.com/oneo-me/nsgo-agio"
	"github.com/oneo-me/nsgo-agio/chatview"
	"github.com/oneo-me/nsgo-agio/transcript"

	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// assistant gives each document with a file a conversation of
// Options.Agents about it, which starts its agent with the first message.
// The agent edits the file itself, as the app's skill teaches, and asks
// the app's Server what the user sees of it.
type assistant struct {
	cx *core.Scope
	e  *Editor
	m  *chatview.Model
}

func newAssistant(cx *core.Scope, e *Editor) *assistant {
	a := &assistant{cx: cx, e: e, m: e.opts.Agents}
	if a.m == nil {
		return a
	}
	// The document shown gets its conversation once it has a file.
	cx.Effect(func() {
		d := e.cur()
		if path := d.path.Get(); path != "" && d.agent.Get() == "" {
			cx.Untrack(func() { a.create(d) })
		}
	})
	return a
}

// prompt is the system prompt of a conversation about the file at path.
func prompt(path string) string {
	return "You are an app designer working in Scenra, a design and flow editor. The user sees the Scenra file " + path + ` beside this conversation, which is about it. Edit it as the scenra skill says, and ask the scenra MCP server's ` + ToolView + ` tool what the user sees of it and has selected before you change it.

Discuss the app with the user: ask about its screens, what each shows, the steps, decisions, actors and edge cases of what happens on them when they are unclear, suggest improvements, and draw as you agree. Reply in the user's language.`
}

// create starts d's conversation, in the folder holding its file.
func (a *assistant) create(d *document) {
	if d.creating {
		return
	}
	d.creating = true
	path := d.path.GetUntracked()
	a.m.Create(agio.Create{Dir: filepath.Dir(path), Title: filepath.Base(path), SystemPrompt: prompt(path)}, func(id string) {
		d.creating = false
		d.agent.Set(id)
	})
}

// remove forgets d's conversation and stops its agent.
func (a *assistant) remove(d *document) {
	id := d.agent.GetUntracked()
	if a.m == nil || id == "" {
		return
	}
	d.agent.Set("")
	a.m.Do(func(b agio.Backend) error { return b.Remove(id) }, nil)
}

// clear starts a new conversation about the document shown.
func (a *assistant) clear() { a.remove(a.e.shown()) }

// ask sends prompt to the conversation about the document shown, saving
// it first; nil without agents.
func (a *assistant) ask() func(prompt string) {
	if a.m == nil {
		return nil
	}
	return func(prompt string) {
		d := a.e.shown()
		id := d.agent.GetUntracked()
		if id == "" || a.e.locked(d) {
			return
		}
		a.sent(d)
		a.m.Do(func(b agio.Backend) error { return b.Send(id, agio.Message{Text: prompt}) }, nil)
	}
}

// sent saves d, so what its agent reads is what the user sees, as a
// message about it is sent.
func (a *assistant) sent(d *document) {
	a.e.saveQuietly(d)
	if a.e.opts.OnSend != nil {
		a.e.opts.OnSend()
	}
}

// working says d's agent is at work, while which the user cannot edit d;
// a caller reading it follows.
func (a *assistant) working(d *document) bool {
	if a.m == nil {
		return false
	}
	id := d.agent.Get()
	return id != "" && a.m.Conversation(id).Running
}

// busy is working as the backend has it now, which the model follows a
// little later, for actions to check.
func (a *assistant) busy(d *document) bool {
	if a.m == nil {
		return false
	}
	c := a.m.Backend().State().Conversation(d.agent.GetUntracked())
	return c != nil && c.Running
}

// group is the undo group of a change to d the agent made now: its
// running turn, whose changes undo together, else none.
func (a *assistant) group(d *document) string {
	if a.m == nil {
		return ""
	}
	// The backend's state, as the model's follows it a little later.
	c := a.m.Backend().State().Conversation(d.agent.GetUntracked())
	if c == nil || !c.Running || len(c.Turns) == 0 {
		return ""
	}
	t := c.Turns[len(c.Turns)-1]
	if t.Status != transcript.Running {
		return ""
	}
	return c.ID + "/" + t.ID
}

// assistantPane is the conversation about the document shown under a
// header that is the window's title bar with titleBar; or, while the
// document has no file, a way to make or open one.
func assistantPane(cx *core.Scope, a *assistant, titleBar bool) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	e := a.e
	_, drag := chrome(cx, titleBar)
	actions := []core.Node{ui.Spacer{},
		ui.IconButton{Icon: lucide.MessageSquarePlus, Label: tr.NewChat(), OnPress: a.clear, TestID: "new-chat"}}
	if open := e.opts.OpenSettings; open != nil {
		actions = append(actions, ui.IconButton{Icon: lucide.Settings, Label: tr.Settings(), OnPress: open, TestID: "open-settings"})
	}
	header := core.View{Desktop: core.DesktopViewProps{WindowDrag: drag},
		Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8, FlexShrink: 0, Borders: core.Edges{Bottom: 1}},
		StyleFunc: func() core.Style {
			bar, _ := chrome(cx, titleBar)
			return core.Style{Height: core.Pt(max(HeaderHeight, bar.Top)), Padding: core.Edges{Left: 12, Right: 8 + bar.Right},
				Background: ui.Chrome(cx), BorderColors: core.EdgeColors{Bottom: colors.Line()}}
		},
		Children: append([]core.Node{
			ui.Icon(lucide.Sparkles, 16, colors.Muted),
			ui.Text(tr.Assistant(), 13, ui.Semibold, colors.Text),
		}, actions...)}
	placeholder := func(detail string, actions ...core.Node) core.Node {
		return core.View{Style: core.Style{FlexGrow: 1, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Padding: core.All(32), Gap: 10},
			Children: []core.Node{
				ui.Icon(lucide.Workflow, 32, colors.Muted),
				ui.Text(tr.EmptyTitle(), 15, ui.Semibold, colors.Text),
				core.Text{Content: detail, Style: core.Style{FontSize: 13, LineHeight: 19, TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(320)},
					StyleFunc: ui.Colored(colors.Muted)},
				core.View{Style: core.Style{Direction: core.Row, Gap: 8, Margin: core.Edges{Top: 4}}, Children: actions},
			}}
	}
	// The conversation is built again for another one, as its draft and
	// scrolling are its own.
	body := core.Dynamic(func(cx *core.Scope) core.Node {
		id := e.cur().agent.Get()
		if a.m == nil || id == "" {
			return placeholder(tr.NoFiles(),
				ui.Button{Text: tr.NewFlow(), Icon: lucide.FilePlus, Variant: ui.ButtonPrimary, OnPress: e.NewFile, TestID: "chat-new-file"},
				ui.Button{Text: tr.Open(), Icon: lucide.FolderOpen, Variant: ui.ButtonOutline, OnPress: e.Open, TestID: "chat-open-file"})
		}
		empty := func() bool {
			c := a.m.Conversation(id)
			return len(c.Turns) == 0 && !c.Loading
		}
		// The placeholder leaves the composer free.
		return core.View{Style: ui.FillRest, Children: []core.Node{
			// What the user changed is saved for the agent to read.
			chatview.Conversation{Model: a.m, ID: id, TestID: "conversation", OnSend: func() { a.sent(e.shown()) }},
			core.Show{When: empty, Then: func(*core.Scope) core.Node {
				return core.View{NoFocus: true, Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(180)},
					Children: []core.Node{placeholder(tr.EmptyDetail())}}
			}},
		}}
	})
	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{header, body}}
}
