package scenraview

import (
	"strings"
	"testing"
	"time"
)

func TestLongConversationKeepsTheComposer(t *testing.T) {
	long := strings.Repeat("A long line of the reply. ", 20)
	app, _, d := startAssistant(t)
	newFileAt(t, app, "long.scenra")
	app.Press("id:chat-new-file")
	for i := range 6 {
		d.script(reply{text: strings.Repeat(long+"\n\n", 6)})
		send(app, "Question")
		app.WaitUntil(5*time.Second, func() bool { return len(app.Query("text:Question")) == i+1 && app.Exists("role:button Send") })
		app.Settle()
	}
	// The transcript scrolls and the composer stays in the window.
	var bottom float64
	for v := app.Query("id:composer")[0]; v != nil; v = v.Parent {
		bottom += float64(v.Frame.Origin.Y)
	}
	bottom += float64(app.Query("id:composer")[0].Frame.Size.Height)
	if bottom > 800 {
		t.Fatalf("the composer ends at %v, below the window\n%s", bottom, app.Dump())
	}
}
