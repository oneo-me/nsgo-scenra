package scenraview

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"

	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// TestMain gives the platform the main thread under nsgo test -target.
func TestMain(m *testing.M) { nsgotest.Main(m) }

// The storage keys of the files the test app keeps open.
const (
	keyOpenFiles  = "editor.openFiles"
	keyActiveFile = "editor.activeFile"
)

// newFileAt answers the next save dialogs with path, in a new folder.
func newFileAt(t *testing.T, app *nsgotest.App, name string) string {
	path := filepath.Join(t.TempDir(), name)
	app.SaveFile = func(core.SaveRequest) string { return path }
	return path
}

// writeFlow writes a flow file named name in a new folder.
func writeFlow(t *testing.T, name, text string) string {
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// startEditor starts an app of the Window, whose assistant's agent is
// driver, and returns its editor.
func startEditor(t *testing.T, opts ...nsgotest.Option) (*nsgotest.App, *Editor) {
	app, e, _ := startAssistant(t, opts...)
	return app, e
}

// tabNames are the names of the open tabs, in order.
func tabNames(app *nsgotest.App) []string {
	var names []string
	for _, v := range app.Query("id:title-bar >> role:tab") {
		names = append(names, v.Props.Label)
	}
	return names
}

func TestOpensFlowFilesInTabs(t *testing.T) {
	plan := writeFlow(t, "plan.scenra", "title \"Plan\"\nnode a \"Research\" note=\"Read the docs\"\na -> b\n")
	shop := writeFlow(t, "shop.scenra", "screen home \"Home\"\n  text \"Welcome\"\n")
	app, e := startEditor(t)
	open := func(path string) {
		app.Post(func() {
			if err := e.OpenPath(path); err != nil {
				t.Error(err)
			}
		})
		app.Settle()
	}
	open(plan)
	app.Find("flow >> label:Research")
	app.Find("flow >> text:Read the docs")
	if got := tabNames(app); !slices.Equal(got, []string{"plan.scenra"}) {
		t.Fatalf("tabs %v", got)
	}
	open(shop)
	app.WaitFor("design >> text:Welcome", time.Second)
	// A file open already shows its tab again.
	open(plan)
	app.WaitFor("flow >> label:Research", time.Second)
	if got := tabNames(app); !slices.Equal(got, []string{"plan.scenra", "shop.scenra"}) {
		t.Fatalf("tabs %v", got)
	}
	// The sidebar lists the entries of the document shown.
	app.Press("id:title-bar >> label:shop.scenra")
	app.WaitFor("id:design-home", time.Second)
	if app.Exists("id:flow-main") {
		t.Fatal("the sidebar still lists the other document's flow")
	}

	// The files open again at the next start, the one shown too.
	stored, _ := app.Storage().GetItem(keyOpenFiles)
	again, _ := startEditor(t, nsgotest.Storage(map[string]string{keyOpenFiles: stored, keyActiveFile: shop}))
	if got := tabNames(again); !slices.Equal(got, []string{"plan.scenra", "shop.scenra"}) {
		t.Fatalf("restored tabs %v", got)
	}
	again.Find("design >> text:Welcome")
}

func TestDesignSelectsWhatThePointerIsOver(t *testing.T) {
	shop := writeFlow(t, "shop.scenra", "screen home \"Home\"\n"+
		"  box class=\"flex-1 gap-4 p-6\"\n"+
		"    text \"Welcome\" class=\"text-2xl\"\n"+
		"    box class=\"flex-row gap-2 p-2\"\n"+
		"      icon mail class=\"size-5\"\n"+
		"      text \"Mail\"\n"+
		"    end\n"+
		"  end\n")
	app, e := startEditor(t)
	app.Post(func() {
		if err := e.OpenPath(shop); err != nil {
			t.Error(err)
		}
	})
	app.WaitFor("design >> text:Mail", time.Second)
	// The design fits the pane scaled down, which the layer over it
	// undoes: the pointer is where the camera drew each element.
	pane, design := app.Find("id:design-pointer").Frame.Size, app.Find("id:design").Frame.Size
	zoom := min(1, (pane.Width-2*designFitMargin)/design.Width, (pane.Height-2*designFitMargin)/design.Height)
	if zoom >= 1 {
		t.Fatalf("the design shows unscaled at %v in %v", design, pane)
	}
	at := func(path string) core.Point {
		v := app.Find("id:element-" + path)
		x, y := v.Frame.Size.Width/2, v.Frame.Size.Height/2
		for ; v.Props.TestID != "design"; v = v.Parent {
			x, y = x+v.Frame.Origin.X, y+v.Frame.Origin.Y
		}
		return core.Point{X: (pane.Width-design.Width*zoom)/2 + x*zoom, Y: (pane.Height-design.Height*zoom)/2 + y*zoom}
	}
	app.HoverAt("id:design-pointer", at("0.1.0"))
	app.WaitFor("id:design-hovered", time.Second)
	app.PressAt("id:design-pointer", at("0.1.0"))
	if got := e.cur().element.GetUntracked(); got != "0.1.0" {
		t.Fatalf("pressing the icon selected %q", got)
	}
	app.WaitFor("id:design-selected", time.Second)
	// The comment card opens beside where the pointer pressed, and typing
	// in it keeps the element selected.
	card, pressed := app.Find("id:comment-card").Frame, at("0.1.0")
	if dx, dy := card.Origin.X-pressed.X, card.Origin.Y-pressed.Y; dx < 0 || dx > 20 || dy < 0 || dy > 20 {
		t.Fatalf("the card is at %v, pressed at %v", card.Origin, pressed)
	}
	app.Type("id:comment-input", "Bigger")
	if got := e.cur().element.GetUntracked(); got != "0.1.0" || !app.Exists("id:comment-card") {
		t.Fatalf("typing in the card selected %q", got)
	}
	app.HoverAt("id:design-pointer", at("0.0"))
	app.PressAt("id:design-pointer", at("0.0"))
	if got := e.cur().element.GetUntracked(); got != "0.0" {
		t.Fatalf("pressing the title selected %q", got)
	}
	app.HoverOut()
	if app.Exists("id:design-hovered") {
		t.Fatal("the outline stayed after the pointer left")
	}
	app.PressAt("id:design-pointer", core.Point{X: 4, Y: 4})
	if got := e.cur().element.GetUntracked(); got != "" {
		t.Fatalf("pressing around the design left %q selected", got)
	}
}

func TestSavesWhereTheFileWasSaved(t *testing.T) {
	app, e := startEditor(t)
	// A new file is saved where the user chooses before it opens.
	saveTo := newFileAt(t, app, "chosen.scenra")
	app.Press("id:new-file")
	app.Settle()
	if got := tabNames(app); !slices.Equal(got, []string{"chosen.scenra"}) {
		t.Fatalf("tabs after making a file %v", got)
	}
	if _, err := os.Stat(saveTo); err != nil {
		t.Fatal(err)
	}
	retitle := func(title string) {
		app.Post(func() {
			d := e.shown()
			p := d.doc.GetUntracked()
			p.Title = title
			d.change(p, "")
		})
		app.Settle()
	}
	// Saving writes the same file without asking where.
	retitle("Second")
	app.Post(e.Save)
	app.Settle()
	if len(app.Saves) != 1 {
		t.Fatalf("asked where to save %d times", len(app.Saves))
	}
	if data, _ := os.ReadFile(saveTo); !strings.Contains(string(data), "Second") {
		t.Fatalf("the file was not saved:\n%s", data)
	}

	// An opened file saves where it came from.
	opened := writeFlow(t, "opened.scenra", "title \"Opened\"\n")
	app.Post(func() {
		if err := e.OpenPath(opened); err != nil {
			t.Error(err)
		}
	})
	app.Settle()
	retitle("Changed")
	app.Post(e.Save)
	app.Settle()
	if data, _ := os.ReadFile(opened); len(app.Saves) != 1 || !strings.Contains(string(data), "Changed") {
		t.Fatalf("saves %d, file:\n%s", len(app.Saves), data)
	}

	// Canceling the dialog makes no file.
	app.SaveFile = nil
	app.Post(e.NewFile)
	app.Settle()
	if got := tabNames(app); len(got) != 2 {
		t.Fatalf("tabs after canceling %v", got)
	}
}

func TestUnsavedChangesAskBeforeTheyGo(t *testing.T) {
	path := writeFlow(t, "plan.scenra", "title \"Plan\"\nnode a \"Research\"\n")
	app, e := startEditor(t, nsgotest.PendingAlerts())
	open := func() {
		app.Post(func() {
			if err := e.OpenPath(path); err != nil {
				t.Error(err)
			}
		})
		app.Settle()
	}
	retitle := func(title string) {
		app.Post(func() {
			d := e.shown()
			p := d.doc.GetUntracked()
			p.Title = title
			d.change(p, "")
		})
		app.Settle()
	}
	closeTab := func() {
		app.Post(e.CloseShown)
		app.Settle()
		if alerts := app.Alerts(); len(alerts) != 1 || !strings.Contains(alerts[0].Title, "plan.scenra") {
			t.Fatalf("alerts %+v", alerts)
		}
	}

	// Canceling keeps the tab.
	open()
	retitle("Edited plan")
	closeTab()
	app.Answer(2)
	app.Settle()
	if got := tabNames(app); !slices.Equal(got, []string{"plan.scenra"}) {
		t.Fatalf("after canceling, the tabs are %v", got)
	}

	// Saving writes the file before the tab closes.
	closeTab()
	app.Answer(0)
	app.Settle()
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), "Edited plan") {
		t.Fatalf("the file was not saved:\n%s", data)
	}
	if got := tabNames(app); len(got) != 0 {
		t.Fatalf("after closing the last tab, the tabs are %v", got)
	}

	// Not saving drops the changes.
	open()
	retitle("Dropped")
	closeTab()
	app.Answer(1)
	app.Settle()
	if data, _ := os.ReadFile(path); strings.Contains(string(data), "Dropped") {
		t.Fatal("not saving saved")
	}

	// A clean document goes without asking.
	app.Post(e.CloseShown)
	app.Settle()
	if len(app.Alerts()) != 0 {
		t.Fatalf("a clean document asked: %+v", app.Alerts())
	}
}

func must(name string, ok bool) string {
	if !ok {
		return ""
	}
	return name
}

func TestEveryIconNameDraws(t *testing.T) {
	for _, name := range append(slices.Clone(flowfile.CommonIcons), "home", "edit", "alert-circle", "bar-chart-2") {
		if lucideIcons[must(flowfile.IconName(name))] == nil {
			t.Errorf("no icon %q", name)
		}
	}
	for name := range lucideIcons {
		if _, ok := flowfile.IconName(name); !ok {
			t.Errorf("flow files refuse icon %q", name)
		}
	}
}

func TestExamplesDrawEveryClassAndDrawing(t *testing.T) {
	paths, _ := filepath.Glob("../examples/*.scenra")
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		p, err := flowfile.ParseProject(string(data))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if unknown := unknownClasses(p); len(unknown) > 0 {
			t.Errorf("%s: classes that draw nothing: %v", path, unknown)
		}
		var walk func(els []flowfile.Element)
		walk = func(els []flowfile.Element) {
			for _, el := range els {
				if el.Kind == "svg" {
					if _, err := core.ParseSVG([]byte(el.Text)); err != nil {
						t.Errorf("%s: %v", path, err)
					}
				}
				walk(el.Children)
			}
		}
		for _, s := range p.Screens {
			walk(s.UI)
		}
	}
}

func TestAnotherAppShowsOneFile(t *testing.T) {
	path := writeFlow(t, "shop.scenra", "screen home \"Home\"\n  text \"Welcome\"\nflow main \"Main\"\n  node a \"Browse\"\n")
	var e *Editor
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		e = New(cx, Options{})
		if err := e.OpenPath(path); err != nil {
			t.Error(err)
		}
		return core.View{Style: ui.FillRest.Merge(core.Style{Direction: core.Row}), Children: []core.Node{Sidebar{e}, Workspace{e}}}
	}, nsgotest.Size(1000, 700))
	app.Find("design >> text:Welcome")
	app.Press("id:flow-main")
	app.WaitFor("flow >> label:Browse", time.Second)
	if app.Exists("id:title-bar") || app.Exists("id:composer") {
		t.Fatal("the views shown brought the tabs or the assistant")
	}
	if _, ok := app.Storage().GetItem(keyOpenFiles); ok {
		t.Fatal("an editor that remembers nothing kept its files")
	}
}
