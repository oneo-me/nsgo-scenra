package scenraview

import (
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestStillDrawsAFilesDesignsWithItsProjectsComponents(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"components/badge.scenra": "component\ntext \"Badge text\"\n",
	})
	source := "design device=phone\nuse badge\ntext \"Orders\"\n"
	if !Drawable(source) || Drawable("flow\n") || Drawable("design\n  oops") {
		t.Fatal("Drawable tells designs from what Still cannot draw")
	}
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Still{Source: source, Project: dir, TestID: "still"}
	})
	app.WaitFor("id:still >> text:Orders", 5*time.Second)
	app.WaitFor("id:still >> text:Badge text", 5*time.Second)
	if w := app.Find("id:still").Frame.Size.Width; w < 390 {
		t.Fatalf("the phone design is %v wide", w)
	}
}
