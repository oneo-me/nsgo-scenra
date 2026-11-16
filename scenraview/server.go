package scenraview

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	agentio "github.com/oneo-me/nsgo-agio/agent"
	"github.com/oneo-me/nsgo-agio/mcp"
)

// ServerName names the editor's MCP server to agents.
const ServerName = "scenra"

// ToolView is what the user sees of the open folder, and ToolLayout
// where the elements of a file's designs or component lay out.
const (
	ToolView   = "get_view"
	ToolLayout = "get_layout"
)

// callLimit is how long a call waits for the editor to lay designs out
// before it answers without them.
const callLimit = 10 * time.Second

// Server tells agents over MCP what the user sees of the folder open in
// an editor, as agents edit its files themselves but cannot see the
// editor.
// An app starts it before its window, to give the agents it runs, and
// attaches the editor once there is one; calls fail while none is
// attached.
type Server struct {
	srv *mcp.Server
	mu  sync.Mutex
	e   *Editor
}

// Listen starts a server on the loopback interface.
func Listen() (*Server, error) {
	s := &Server{}
	srv, err := mcp.Listen(mcp.Options{Name: ServerName, Tools: []mcp.Tool{{Name: ToolView,
		Description: "Tell what the user sees in Scenra: the open folder, the design or flow open in it, the node or element selected and the user's comments, and the problems of the folder's Scenra files: those that do not parse, the classes that draw nothing, and how their designs lay out wrong, such as text cut off, elements outside their design or of no size, and w-full filling nothing. Call it before changing the files and after writing them.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}}}, {Name: ToolLayout,
		Description: "Lay out a Scenra file's designs, or its component, as the editor draws them, and tell where each element went: its path, kind, x and y from the design's top left corner, width and height, element by element, and for a use the component and variants it draws. Paths inside a component follow a colon, and those of a design an outlet shows a bar.",
		InputSchema: map[string]any{"type": "object", "required": []string{"path"}, "properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "The file's path in the open folder, such as main.scenra or components/tab.scenra"},
			"design":  map[string]any{"type": "string", "description": "The ID of one design of a file of several; all of them when left out"},
			"variant": map[string]any{"type": "string", "description": "The ID of the component's variant to draw it as; the component itself when left out"},
		}}}},
		Call: s.call})
	if err != nil {
		return nil, err
	}
	s.srv = srv
	return s, nil
}

// Agent is how agents reach the server, for host.Options.
func (s *Server) Agent() agentio.MCPServer { return s.srv.Agent() }

// Attach answers the calls from e from now on, until e's scope ends.
func (s *Server) Attach(e *Editor) {
	s.mu.Lock()
	s.e = e
	s.mu.Unlock()
	e.cx.Cleanup(func() {
		s.mu.Lock()
		if s.e == e {
			s.e = nil
		}
		s.mu.Unlock()
	})
}

// Close stops the server.
func (s *Server) Close() error { return s.srv.Close() }

// call answers a call on the UI's goroutine and waits for it.
func (s *Server) call(ctx context.Context, name string, arguments json.RawMessage) (string, bool) {
	s.mu.Lock()
	e := s.e
	s.mu.Unlock()
	if e == nil {
		return "error: no Scenra editor is open", true
	}
	type answer struct {
		text   string
		failed bool
	}
	done := make(chan answer, 2)
	switch name {
	case ToolLayout:
		var args struct{ Path, Design, Variant string }
		if err := json.Unmarshal(arguments, &args); err != nil || args.Path == "" {
			return "error: " + ToolLayout + " takes the path of a file", true
		}
		e.cx.Post(func() {
			e.layoutOf(args.Path, args.Design, args.Variant, func(text string, failed bool) { done <- answer{text, failed} })
		})
	default:
		e.cx.Post(func() {
			// The view without the layout answers if laying out takes too
			// long.
			var timer *time.Timer
			e.viewLaidOut(func(quick string) {
				timer = time.AfterFunc(callLimit, func() { done <- answer{quick, false} })
			}, func(text string) {
				timer.Stop()
				done <- answer{text, false}
			})
		})
	}
	select {
	case a := <-done:
		return a.text, a.failed
	case <-ctx.Done():
		return "error: " + ctx.Err().Error(), true
	}
}
