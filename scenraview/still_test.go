package scenraview

import (
	"path/filepath"
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
	if !Drawable(source) || Drawable("design\n  oops") {
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

func TestStillDrawsAFilesFlowsAsTheEditorDoes(t *testing.T) {
	source := "flow pay name=\"Pay\" direction=right\nnode a \"Cart\" note=\"Review the order\"\nnode b \"Paid\" kind=end\na -> b \"Confirm\"\n\n" +
		"flow refund name=\"Refund\"\nnode c \"Request\"\n"
	if !Drawable(source) || !Drawable("flow\n") {
		t.Fatal("Drawable leaves out flows")
	}
	dir := folderOfFiles(t, map[string]string{"flows/pay.scenra": source})
	still := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Still{Source: source, Project: dir, TestID: "still"}
	})
	still.WaitFor("id:still >> id:flow-0 >> text:Cart", 5*time.Second)
	still.Find("id:still >> id:flow-0 >> text:Review the order")
	still.Find("id:still >> id:flow-1 >> text:Request")
	still.Find("id:still >> text:Refund")
	still.Advance(time.Second)
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "flows", "pay.scenra"))
	app.WaitFor("id:flows >> text:Cart", 5*time.Second)
	app.Advance(time.Second)
	if got, want := still.Find("id:still >> id:flows").Frame.Size, app.Find("id:flows").Frame.Size; got != want || got.Width == 0 {
		t.Fatalf("the flows take %v, not %v as in the editor", got, want)
	}
	for _, id := range []string{"flow-0", "flow-1"} {
		if got, want := still.Find("id:still >> id:"+id).Frame, app.Find("id:flows >> id:"+id).Frame; got != want {
			t.Fatalf("%s is at %v, not %v as in the editor", id, got, want)
		}
	}
}

func TestShowcaseDrawsAProjectsComponentsAsTheEditorDoes(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra":                "design\nuse tab-bar\n",
		"components/tab-bar.scenra":  "component name=\"Tab bar\"\nbox class=\"flex-row\"\n  use nav-item\n  text \"Orders\"\nend\n",
		"components/nav-item.scenra": "component name=\"Navigation/Item\" width=200\ntext \"Home tab\"\n",
		"components/switch.scenra": "component backdrop=\"bg-inset p-1\"\nbox name=track class=\"h-6 w-10 rounded-full bg-zinc-300\"\nend\n" +
			"variant on track.class=\"bg-blue-600\"\n",
	})
	still := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Showcase{Project: dir, TestID: "still"}
	})
	still.WaitFor("id:still >> id:showcase-tab-bar >> text:Home tab", 5*time.Second)
	still.Find("id:still >> id:showcase-group-Navigation")
	still.Find("id:still >> id:showcase-drawing-switch >> id:variant-on")
	if still.Exists("id:still >> text:Welcome") {
		t.Fatal("the showcase draws a design")
	}
	still.Advance(time.Second)
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	app.WaitFor("id:sidebar-components", 5*time.Second)
	app.Press("id:sidebar-components")
	app.WaitFor("id:showcase-tab-bar", 5*time.Second)
	app.Advance(time.Second)
	if got, want := still.Find("id:still >> id:showcase").Frame.Size, app.Find("id:showcase").Frame.Size; got != want {
		t.Fatalf("the showcase takes %v, not %v as in the editor", got, want)
	}
	for _, id := range []string{"tab-bar", "nav-item", "switch"} {
		if got, want := still.Find("id:still >> id:showcase-"+id).Frame, app.Find("id:showcase-"+id).Frame; got != want {
			t.Fatalf("%s is at %v, not %v as in the editor", id, got, want)
		}
	}
	// A project without components draws none.
	empty := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Showcase{Project: folderOfFiles(t, map[string]string{"home.scenra": "design\n"}), TestID: "still"}
	})
	empty.Advance(time.Second)
	if empty.Exists("id:still >> id:showcase") {
		t.Fatal("an empty project draws a showcase")
	}
}
