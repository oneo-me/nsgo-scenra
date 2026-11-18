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

func TestStillDrawsADesignFillingAnOutletInItsHost(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"main.scenra": "design device=desktop\nbox class=\"flex-1 flex-row\"\n  text \"Sidebar\"\n  box class=\"flex-1\" outlet=content\n  end\nend\n",
	})
	// This version of the view is drawn, not the one the folder holds.
	source := "design in=\"main#content\"\ntext \"Orders\"\n"
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Still{Source: source, Project: dir, TestID: "still"}
	})
	app.WaitFor("id:still >> text:Sidebar", 5*time.Second)
	app.WaitFor("id:still >> text:Orders", 5*time.Second)
	// Where the host is not there, it draws alone.
	alone := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Still{Source: "design in=\"gone#content\"\ntext \"Alone\"\n", Project: dir, TestID: "still"}
	})
	alone.WaitFor("id:still >> text:Alone", 5*time.Second)
}
