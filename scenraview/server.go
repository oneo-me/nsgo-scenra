package scenraview

import (
	"context"
	"encoding/json"
	"sync"

	agentio "github.com/oneo-me/nsgo-agio/agent"
	"github.com/oneo-me/nsgo-agio/mcp"
)

// ServerName names the editor's MCP server to agents.
const ServerName = "scenra"

// ToolView is the server's one tool: what the user sees of the open
// folder.
const ToolView = "get_view"

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
		Description: "Tell what the user sees in Scenra: the open folder, the design or flow open in it, the node or element selected and the user's comments, and the problems of the folder's Scenra files: those that do not parse and the classes that draw nothing. Call it before changing the files and after writing them.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}}}},
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
	done := make(chan string, 1)
	e.cx.Post(func() { done <- e.viewOf() })
	select {
	case text := <-done:
		return text, false
	case <-ctx.Done():
		return "error: " + ctx.Err().Error(), true
	}
}
