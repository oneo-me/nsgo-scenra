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

// The storage keys of the folder and files the test app keeps open.
const (
	keyFolder     = "editor.folder"
	keyOpenFiles  = "editor.openFiles"
	keyActiveFile = "editor.activeFile"
)

// folderOfFiles writes files, by their paths with slashes, in a new
// folder.
func folderOfFiles(t *testing.T, files map[string]string) string {
	dir := t.TempDir()
	for name, text := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// writeFlow writes a Scenra file named name in a new folder.
func writeFlow(t *testing.T, name, text string) string {
	return filepath.Join(folderOfFiles(t, map[string]string{name: text}), name)
}

// openPath opens the folder or file at path in e.
func openPath(t *testing.T, app *nsgotest.App, e *Editor, path string) {
	app.Post(func() {
		if err := e.OpenPath(path); err != nil {
			t.Error(err)
		}
	})
	app.Settle()
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

func TestShowsTheFolderAsATree(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"plan.scenra":             "flow\nnode a \"Research\" note=\"Read the docs\"\na -> b\n",
		"screens/home.scenra":     "design\ntext \"Welcome\"\n",
		"screens/broken.scenra":   "box\n",
		".hidden/secret.scenra":   "flow\n",
		"node_modules/pkg.scenra": "flow\n",
		"notes.txt":               "design\n",
		"empty/nothing-here.md":   "",
	})
	app, e := startEditor(t)
	app.Find("id:sidebar-open-folder")
	openPath(t, app, e, dir)
	app.WaitFor("id:files >> label:plan", time.Second)
	if got := app.Find("id:folder-name").Text; got != filepath.Base(dir) {
		t.Fatalf("the folder is named %q", got)
	}
	for _, name := range []string{"screens", "home", "broken"} {
		app.Find("id:files >> label:" + name)
	}
	for _, name := range []string{"secret", "pkg", "notes", "empty"} {
		if app.Exists("id:files >> label:" + name) {
			t.Errorf("the tree shows %s", name)
		}
	}
	// Choosing a file shows it in a tab.
	app.Press("id:files >> label:plan")
	app.WaitFor("flow >> label:Research", time.Second)
	app.Find("flow >> text:Read the docs")
	app.Press("id:files >> label:home")
	app.WaitFor("design >> text:Welcome", time.Second)
	if got := tabNames(app); !slices.Equal(got, []string{"plan.scenra", "home.scenra"}) {
		t.Fatalf("tabs %v", got)
	}
	app.Press("id:title-bar >> label:plan.scenra")
	app.WaitFor("flow >> label:Research", time.Second)
	// A file another program makes shows in the tree.
	os.WriteFile(filepath.Join(dir, "screens", "later.scenra"), []byte("flow\n"), 0o644)
	app.WaitFor("id:files >> label:later", 5*time.Second)

	// The folder and its files open again at the next start, the one
	// shown too.
	stored, _ := app.Storage().GetItem(keyOpenFiles)
	again, _ := startEditor(t, nsgotest.Storage(map[string]string{keyFolder: dir, keyOpenFiles: stored,
		keyActiveFile: filepath.Join(dir, "screens", "home.scenra")}))
	if got := tabNames(again); !slices.Equal(got, []string{"plan.scenra", "home.scenra"}) {
		t.Fatalf("restored tabs %v", got)
	}
	again.Find("design >> text:Welcome")
	again.WaitFor("id:files >> label:plan", time.Second)
}

func TestOpeningAFileOpensItsFolder(t *testing.T) {
	one := folderOfFiles(t, map[string]string{"a.scenra": "flow\nnode a \"First\"\n"})
	other := writeFlow(t, "b.scenra", "flow\nnode b \"Second\"\n")
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(one, "a.scenra"))
	app.WaitFor("flow >> label:First", time.Second)
	if got := core.Untracked(e.cx, e.Folder); got != one {
		t.Fatalf("folder %q, want %q", got, one)
	}
	openPath(t, app, e, other)
	app.WaitFor("flow >> label:Second", time.Second)
	if got := tabNames(app); !slices.Equal(got, []string{"b.scenra"}) || core.Untracked(e.cx, e.Folder) != filepath.Dir(other) {
		t.Fatalf("tabs %v in %s", got, core.Untracked(e.cx, e.Folder))
	}
	app.WaitFor("id:files >> label:b", time.Second)
	if app.Exists("id:files >> label:a") {
		t.Fatal("the tree still shows the other folder")
	}
}

func TestDesignSelectsWhatThePointerIsOver(t *testing.T) {
	shop := writeFlow(t, "shop.scenra", "design\n"+
		"box class=\"flex-1 gap-4 p-6\"\n"+
		"  text \"Welcome\" class=\"text-2xl\"\n"+
		"  box class=\"flex-row gap-2 p-2\"\n"+
		"    icon mail class=\"size-5\"\n"+
		"    text \"Mail\"\n"+
		"  end\n"+
		"end\n")
	app, e := startEditor(t)
	openPath(t, app, e, shop)
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

func TestMakesFilesInTheFolder(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{"flows/old.scenra": "flow\n"})
	app, e := startEditor(t)
	openPath(t, app, e, dir)
	app.WaitFor("id:files >> label:old", time.Second)
	create := func(item, name string) {
		app.Press("id:new-file")
		app.Press("id:" + item)
		app.WaitFor("id:new-file-dialog", time.Second)
		app.Type("id:new-file-name", name)
		app.Press("id:new-file-create")
		app.Settle()
	}
	// A design is made in the folder.
	create("new-design", "Sign in")
	if data, err := os.ReadFile(filepath.Join(dir, "Sign in.scenra")); err != nil || string(data) != "design\n" {
		t.Fatalf("the design: %q %v", data, err)
	}
	if got := tabNames(app); !slices.Equal(got, []string{"Sign in.scenra"}) {
		t.Fatalf("tabs %v", got)
	}
	app.Find("id:files >> label:Sign in")
	app.WaitForGone("id:new-file-dialog", time.Second)
	// A name in use is refused.
	create("new-design", "Sign in.scenra")
	app.Find("id:new-file-problem")
	app.Press("id:new-file-cancel")
	app.WaitForGone("id:new-file-dialog", time.Second)
	// A flow goes in the folder selected.
	app.Press(`id:files >> text:"flows"`)
	create("new-flow", "checkout")
	if data, err := os.ReadFile(filepath.Join(dir, "flows", "checkout.scenra")); err != nil || string(data) != "flow\n" {
		t.Fatalf("the flow: %q %v", data, err)
	}
	app.Find("id:files >> label:checkout")

	// Saving writes the file the document came from.
	app.Post(func() {
		d := e.shown()
		f := d.doc.GetUntracked()
		f.Flow.Direction = flowfile.Right
		d.change(f, "")
	})
	app.Settle()
	app.Post(e.Save)
	app.Settle()
	if data, _ := os.ReadFile(filepath.Join(dir, "flows", "checkout.scenra")); string(data) != "flow direction=right\n" {
		t.Fatalf("the file was not saved:\n%s", data)
	}
}

func TestUnsavedChangesAskBeforeTheyGo(t *testing.T) {
	path := writeFlow(t, "plan.scenra", "flow\nnode a \"Research\"\n")
	app, e := startEditor(t, nsgotest.PendingAlerts())
	open := func() { openPath(t, app, e, path) }
	relabel := func(label string) {
		app.Post(func() {
			d := e.shown()
			f := d.doc.GetUntracked().Clone()
			f.Flow.Nodes[0].Label = label
			d.change(f, "")
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
	relabel("Edited plan")
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
	relabel("Dropped")
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
			t.Errorf("Scenra files refuse icon %q", name)
		}
	}
}

func TestExamplesDrawEveryClassAndDrawing(t *testing.T) {
	files, _ := scan("../examples", nil)
	if len(files) < 2 {
		t.Fatalf("examples %v", files)
	}
	for _, en := range files {
		path := filepath.Join("../examples", en.path)
		f, err := readFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if unknown := unknownClasses(f.UI); len(unknown) > 0 {
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
		walk(f.UI)
	}
}

func TestAnotherAppShowsOneFile(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{"home.scenra": "design\ntext \"Welcome\"\n", "browse.scenra": "flow\nnode a \"Browse\"\n"})
	var e *Editor
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		e = New(cx, Options{})
		if err := e.OpenPath(filepath.Join(dir, "home.scenra")); err != nil {
			t.Error(err)
		}
		return core.View{Style: ui.FillRest.Merge(core.Style{Direction: core.Row}), Children: []core.Node{Sidebar{e}, Workspace{e}}}
	}, nsgotest.Size(1000, 700))
	app.Find("design >> text:Welcome")
	app.WaitFor("id:files >> label:browse", time.Second)
	app.Press("id:files >> label:browse")
	app.WaitFor("flow >> label:Browse", time.Second)
	if app.Exists("id:title-bar") || app.Exists("id:composer") {
		t.Fatal("the views shown brought the tabs or the assistant")
	}
	if _, ok := app.Storage().GetItem(keyFolder); ok {
		t.Fatal("an editor that remembers nothing kept its folder")
	}
}
