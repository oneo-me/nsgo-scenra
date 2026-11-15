package scenraview

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"github.com/nsgo-dev/nsgo/ui"
	agentio "github.com/oneo-me/nsgo-agio/agent"
	"github.com/oneo-me/nsgo-agio/chatview"
	"github.com/oneo-me/nsgo-agio/host"
	"github.com/oneo-me/nsgo-agio/mcp"
)

// step is what a scripted agent does: write text to file, or, with view,
// ask the server what the user sees of it.
type step struct {
	file, text string
	view       bool
}

// reply is what a scripted agent does for a prompt: its steps, then the
// text it answers once hold, when set, closes.
type reply struct {
	steps []step
	hold  chan struct{}
	text  string
}

// scripted is a fake Claude Code answering each prompt with the next
// reply, keeping the sessions' options, the prompts and what the server
// told it.
type scripted struct {
	mu      sync.Mutex
	replies []reply
	started []agentio.StartOptions
	prompts []string
	views   []string
}

// script queues replies for the prompts to come.
func (d *scripted) script(replies ...reply) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.replies = append(d.replies, replies...)
}

func (d *scripted) get() (started []agentio.StartOptions, prompts, views []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.started), slices.Clone(d.prompts), slices.Clone(d.views)
}

func (d *scripted) Tool() agentio.Tool { return agentio.Claude }

func (d *scripted) Catalog(context.Context, string) (agentio.Catalog, error) {
	return agentio.Catalog{Model: "opus", Models: []agentio.Model{{Tool: agentio.Claude, ID: "opus", Name: "Opus"}}}, nil
}

func (d *scripted) Start(o agentio.StartOptions, onEvent func(agentio.Event), onExit func(error)) (agentio.Session, agentio.Info, error) {
	d.mu.Lock()
	d.started = append(d.started, o)
	d.mu.Unlock()
	return &scriptedSession{d: d, o: o, emit: onEvent}, agentio.Info{Cursor: "session-1", Settings: o.Settings}, nil
}

func (d *scripted) History(string, string) ([]agentio.Event, error) { return nil, nil }

type scriptedSession struct {
	d    *scripted
	o    agentio.StartOptions
	emit func(agentio.Event)
}

func (s *scriptedSession) Prompt(_ context.Context, p agentio.Prompt) error {
	d := s.d
	d.mu.Lock()
	d.prompts = append(d.prompts, p.Text)
	var r reply
	if len(d.replies) > 0 {
		r, d.replies = d.replies[0], d.replies[1:]
	}
	d.mu.Unlock()
	s.emit(agentio.Event{Type: agentio.Started})
	// The steps wait for the UI, so they run once Prompt returns.
	go func() {
		var server agentio.MCPServer
		for _, m := range s.o.MCPServers {
			if m.Name == ServerName {
				server = m
			}
		}
		for _, st := range r.steps {
			if !st.view {
				os.WriteFile(st.file, []byte(st.text), 0o644)
				continue
			}
			out, _, err := mcp.CallTool(context.Background(), server, ToolView, map[string]any{"file": st.file})
			if err != nil {
				out = "error: " + err.Error()
			}
			d.mu.Lock()
			d.views = append(d.views, out)
			d.mu.Unlock()
		}
		if r.hold != nil {
			<-r.hold
		}
		s.emit(agentio.Event{Type: agentio.TextDone, Segment: "text", Text: r.text})
		s.emit(agentio.Event{Type: agentio.Settled})
	}()
	return nil
}

func (s *scriptedSession) Abort(context.Context) error                     { return nil }
func (s *scriptedSession) Configure(context.Context, agentio.Settings) error { return nil }
func (s *scriptedSession) Close(time.Duration)                             {}

// askedSettings counts the times the test app's assistant opened the
// settings.
var askedSettings int

// startAssistant starts an app of the Window whose assistant's agent is a
// scripted one, which asks a Server what the user sees.
func startAssistant(t *testing.T, opts ...nsgotest.Option) (*nsgotest.App, *Editor, *scripted) {
	d := &scripted{}
	srv, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { srv.Close() })
	h := host.New(host.Options{Env: []string{}, Tools: []agentio.Tool{agentio.Claude}, Drivers: map[agentio.Tool]agentio.Driver{agentio.Claude: d},
		Home: t.TempDir(), MCPServers: []agentio.MCPServer{srv.Agent()}})
	t.Cleanup(func() { h.Close() })
	var e *Editor
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		e = New(cx, Options{Agents: chatview.Bind(cx, h), OpenSettings: func() { askedSettings++ }, Remember: "editor"})
		srv.Attach(e)
		return core.View{Style: ui.FillRest, Children: []core.Node{Window{Editor: e}, ui.Toasts{Toaster: e.Toaster()}}}
	}, append([]nsgotest.Option{nsgotest.Size(1280, 800)}, opts...)...)
	app.WaitUntil(5*time.Second, func() bool { return h.State().Ready() })
	return app, e, d
}

// waitComposer waits for the assistant's composer.
func waitComposer(app *nsgotest.App) {
	app.WaitUntil(5*time.Second, func() bool { return len(app.Query("id:composer")) > 0 })
}

// send types text in the assistant's composer and sends it.
func send(app *nsgotest.App, text string) {
	waitComposer(app)
	app.Type("id:composer", text)
	app.Press("role:button Send")
}

func TestAssistantDrawsOnTheChart(t *testing.T) {
	app, e, d := startAssistant(t)
	app.Find("id:chat-new-file")
	asked := askedSettings
	app.Press("id:open-settings")
	if askedSettings != asked+1 {
		t.Fatal("the assistant's settings button did nothing")
	}
	// A conversation is about a file, which the assistant makes.
	path := newFileAt(t, app, "orders.scenra")
	app.Press("id:chat-new-file")
	app.WaitFor("text:Let's design an app together", 5*time.Second)
	hold := make(chan struct{})
	d.script(reply{steps: []step{
		{file: path, view: true},
		{file: path, text: "node order \"Order placed\" kind=start\n"},
		{file: path, view: true},
		{file: path, text: "node order \"Order placed\" kind=start\nnode ship \"Ship order\"\norder -> ship\n"},
		{file: path, view: true},
	}, hold: hold, text: "I drew **two steps**."})
	send(app, "Chart an order flow")
	// What the agent writes shows as it writes it.
	app.WaitFor("flow >> label:Ship order", 5*time.Second)
	app.WaitUntil(5*time.Second, func() bool { _, _, views := d.get(); return len(views) == 3 })
	if _, _, views := d.get(); !strings.Contains(views[0], "no screens yet") || !strings.Contains(views[2], `screen "main" open on flow "main"`) {
		t.Fatalf("views %q", views)
	}
	// The user cannot change the file while the agent works.
	app.WaitUntil(5*time.Second, func() bool { return core.Untracked(e.cx, e.Working) })
	app.Post(e.Undo)
	app.Settle()
	app.Find("flow >> label:Ship order")
	if core.Untracked(e.cx, e.CanUndo) {
		t.Fatal("undo is offered while the agent works")
	}
	close(hold)
	app.WaitFor("text:two steps", 5*time.Second)
	// The agent works where the file is, told of it.
	started, _, _ := d.get()
	if len(started) != 1 || started[0].Dir != filepath.Dir(path) || !strings.Contains(started[0].SystemPrompt, path) || len(started[0].MCPServers) != 1 {
		t.Fatalf("started %+v", started)
	}

	// What the turn wrote undoes together.
	app.WaitUntil(5*time.Second, func() bool { return core.Untracked(e.cx, e.CanUndo) })
	app.Post(e.Undo)
	app.WaitForGone("flow >> label:Order placed", time.Second)

	// A new chat is another conversation.
	app.Press("id:new-chat")
	app.WaitFor("text:Let's design an app together", 5*time.Second)
}

func TestSendingSavesWhatTheUserChanged(t *testing.T) {
	path := writeFlow(t, "shop.scenra", "screen home \"Home\"\n  box class=\"flex-1 gap-4 p-6 bg-nope\"\n    text \"Welcome\"\n  end\n")
	app, e, d := startAssistant(t)
	app.Post(func() {
		if err := e.OpenPath(path); err != nil {
			t.Error(err)
		}
	})
	app.WaitFor("design >> text:Welcome", time.Second)
	app.Press("id:element-0.0")
	app.WaitFor("id:comment-card", time.Second)
	app.Type("id:comment-input", "Friendlier")
	app.Press("id:comment-save")
	app.WaitFor("id:comment-pin-0.0", time.Second)
	if data, _ := os.ReadFile(path); strings.Contains(string(data), "Friendlier") {
		t.Fatal("the comment was saved before a message was sent")
	}
	d.script(reply{steps: []step{{file: path, view: true}}, text: "Seen."})
	send(app, "What do you see?")
	app.WaitFor("text:Seen.", 5*time.Second)
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), `comment="Friendlier"`) {
		t.Fatalf("sending did not save the comment:\n%s", data)
	}
	if core.Untracked(e.cx, e.Edited) {
		t.Fatal("the document still has unsaved changes")
	}
	// The agent hears of the selection, the comment and what draws nothing.
	_, _, views := d.get()
	for _, want := range []string{"selected element 0.0", `screen home, element 0.0: "Friendlier"`, "draw nothing: bg-nope"} {
		if !strings.Contains(views[0], want) {
			t.Fatalf("the view tells nothing of %q:\n%s", want, views[0])
		}
	}
}

func TestAgentsHearOfFilesThatDoNotParse(t *testing.T) {
	path := writeFlow(t, "plan.scenra", "node a \"Research\"\n")
	app, e, d := startAssistant(t)
	app.Post(func() {
		if err := e.OpenPath(path); err != nil {
			t.Error(err)
		}
	})
	app.WaitFor("flow >> label:Research", time.Second)
	d.script(reply{steps: []step{{file: path, text: "node a \"Research\nscreen\n"}, {file: path, view: true}}, text: "Oops."})
	send(app, "Break it")
	app.WaitFor("text:Oops.", 5*time.Second)
	if _, _, views := d.get(); !strings.Contains(views[0], "does not parse") {
		t.Fatalf("views %q", views)
	}
	// The editor keeps the last version that parsed.
	app.Find("flow >> label:Research")
}

func TestCommentsAskTheAssistant(t *testing.T) {
	path := writeFlow(t, "shop.scenra", "screen home \"Home\"\n  box class=\"flex-1 gap-4 p-6\"\n    text \"Welcome\"\n    text \"Mail\" comment=\"Smaller\"\n  end\n")
	app, e, d := startAssistant(t)
	app.Post(func() {
		if err := e.OpenPath(path); err != nil {
			t.Error(err)
		}
	})
	app.WaitFor("design >> text:Welcome", time.Second)
	app.Find("id:comment-pin-0.1")
	app.Find("id:resolve-comments")
	if app.Exists("id:comment-card") {
		t.Fatal("a comment card shows with nothing selected")
	}

	// Commenting on the selected element saves the comment in the file.
	app.Press("id:element-0.0")
	app.WaitFor("id:comment-card", time.Second)
	app.Type("id:comment-input", "Friendlier")
	app.Press("id:comment-save")
	app.WaitFor("id:comment-pin-0.0", time.Second)
	if got := e.shown().doc.GetUntracked().Format(); !strings.Contains(got, `text "Welcome" comment="Friendlier"`) {
		t.Fatalf("the comment was not saved:\n%s", got)
	}
	app.Press("id:comment-remove")
	app.WaitForGone("id:comment-pin-0.0", time.Second)
	app.Type("id:comment-input", "Friendlier")

	// Asking the assistant sends it the comment, which it resolves.
	d.script(reply{steps: []step{{file: path,
		text: "screen home \"Home\"\n  box class=\"flex-1 gap-4 p-6\"\n    text \"Hello there\" class=\"font-bold\"\n    text \"Mail\" comment=\"Smaller\"\n  end\n"}},
		text: "Made it friendlier."})
	waitComposer(app)
	app.Press("id:comment-ask")
	app.WaitFor("text:Made it friendlier.", 5*time.Second)
	if _, prompts, _ := d.get(); len(prompts) != 1 || prompts[0] != "Resolve my comment on element 0.0: Friendlier" {
		t.Fatalf("prompts %q", prompts)
	}
	app.WaitFor("design >> text:Hello there", time.Second)
	app.WaitForGone("id:comment-pin-0.0", time.Second)
	app.Find("id:comment-pin-0.1")
}

func TestServerTellsOfFilesNotOpen(t *testing.T) {
	srv, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { srv.Close() })
	ask := func(file string) (string, bool) {
		out, failed, err := mcp.CallTool(context.Background(), srv.Agent(), ToolView, map[string]any{"file": file})
		if err != nil {
			t.Fatal(err)
		}
		return out, failed
	}
	path := writeFlow(t, "plan.scenra", "screen a \"A\"\n  text \"x\" class=\"bg-nope\"\n")
	if out, failed := ask(path); !failed || !strings.Contains(out, "no Scenra editor") {
		t.Fatalf("a call without an editor: %q", out)
	}
	nsgotest.Start(t, func(cx *core.Scope) core.Node {
		srv.Attach(New(cx, Options{}))
		return core.View{}
	})
	if out, failed := ask(path); failed || !strings.Contains(out, "not open in the editor") || !strings.Contains(out, "bg-nope") {
		t.Fatalf("a file not open: %q", out)
	}
	if out, failed := ask("plan.scenra"); !failed {
		t.Fatalf("a relative path: %q", out)
	}
}
