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

// ToolView is the server's one tool: what the user sees of a file.
const ToolView = "get_view"

// Server tells agents over MCP what the user sees of the files open in an
// editor, as agents edit the files themselves but cannot see the editor.
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
		Description: "Tell what the user sees of a Scenra file in the editor: the screen and flow open, the node or element selected and the user's comments, and the file's problems: whether it parses and the classes that draw nothing. Call it before changing the file and after writing it.",
		InputSchema: map[string]any{"type": "object", "required": []string{"file"}, "properties": map[string]any{
			"file": map[string]any{"type": "string", "description": "Absolute path of the Scenra file."}}}}},
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
	var a struct{ File string }
	json.Unmarshal(arguments, &a)
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
	done := make(chan answer, 1)
	e.cx.Post(func() {
		text, failed := e.viewOf(a.File)
		done <- answer{text, failed}
	})
	select {
	case a := <-done:
		return a.text, a.failed
	case <-ctx.Done():
		return "error: " + ctx.Err().Error(), true
	}
}
