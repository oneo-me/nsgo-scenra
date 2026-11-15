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

// The storage keys of the folder and file the test app keeps open.
const (
	keyFolder = "editor.folder"
	keyFile   = "editor.file"
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

// shownName is the name the header gives the file shown.
func shownName(app *nsgotest.App) string { return app.Find("id:header >> id:file-name").Text }

// treeNames are the names of the tree's rows, in order.
func treeNames(app *nsgotest.App) []string {
	var names []string
	for _, v := range app.Query("id:files >> role:treeitem") {
		names = append(names, v.Props.Label)
	}
	return names
}

func TestShowsTheFolderAsATree(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"plan.scenra":             "flow\nnode a \"Research\" note=\"Read the docs\"\na -> b\n",
		"screens/home.scenra":     "design\ntext \"Welcome\"\n",
		"screens/broken.scenra":   "box\n",
		"screens/cart.scenra":     "design name=\"Shopping cart\"\ntext \"Cart\"\n",
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
	for _, name := range []string{"screens", "home", "broken", "Shopping cart"} {
		app.Find("id:files >> label:" + name)
	}
	for _, name := range []string{"secret", "pkg", "notes", "empty", "cart"} {
		if app.Exists("id:files >> label:" + name) {
			t.Errorf("the tree shows %s", name)
		}
	}
	// Choosing a file shows it in place of the one shown.
	app.Press("id:files >> label:plan")
	app.WaitFor("flow >> label:Research", time.Second)
	app.Find("flow >> text:Read the docs")
	app.Press("id:files >> label:home")
	app.WaitFor("design >> text:Welcome", time.Second)
	if got := shownName(app); got != "home" {
		t.Fatalf("the header names %q", got)
	}
	// A file that names itself shows by that name.
	app.Press("id:files >> label:Shopping cart")
	app.WaitFor("design >> text:Cart", time.Second)
	if got := shownName(app); got != "Shopping cart" {
		t.Fatalf("the header names %q", got)
	}
	app.Press("id:files >> label:home")
	app.WaitFor("design >> text:Welcome", time.Second)
	// A file another program makes shows in the tree.
	os.WriteFile(filepath.Join(dir, "screens", "later.scenra"), []byte("flow\n"), 0o644)
	app.WaitFor("id:files >> label:later", 5*time.Second)

	// The folder and the file shown open again at the next start.
	stored, _ := app.Storage().GetItem(keyFile)
	if stored != filepath.Join(dir, "screens", "home.scenra") {
		t.Fatalf("kept %q", stored)
	}
	again, _ := startEditor(t, nsgotest.Storage(map[string]string{keyFolder: dir, keyFile: stored}))
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
	if got := shownName(app); got != "b" || core.Untracked(e.cx, e.Folder) != filepath.Dir(other) {
		t.Fatalf("%s shown in %s", got, core.Untracked(e.cx, e.Folder))
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
	ask := func(item string) {
		app.Press("id:new-file")
		app.Press("id:" + item)
		app.WaitFor("id:new-file-dialog", time.Second)
	}
	create := func(item, name string) {
		ask(item)
		app.Type("id:new-file-name", name)
		app.Press("id:new-file-create")
		app.Settle()
	}
	// A design is made in the folder.
	create("new-design", "Sign in")
	if data, err := os.ReadFile(filepath.Join(dir, "Sign in.scenra")); err != nil || string(data) != "design\n" {
		t.Fatalf("the design: %q %v", data, err)
	}
	if got := shownName(app); got != "Sign in" {
		t.Fatalf("the header names %q", got)
	}
	app.Find("id:files >> label:Sign in")
	app.WaitForGone("id:new-file-dialog", time.Second)
	// A name in use, or one only a flow has, is refused.
	for _, name := range []string{"Sign in.scenra", "Sign in-flow"} {
		create("new-design", name)
		app.Find("id:new-file-problem")
		app.Press("id:new-file-cancel")
		app.WaitForGone("id:new-file-dialog", time.Second)
	}
	// A new flow is the design's, beside it, which the header switches
	// to.
	ask("new-flow")
	if got := app.Find("id:new-file-name").Text; got != "Sign in" {
		t.Fatalf("the flow is named %q", got)
	}
	app.Press("id:new-file-create")
	app.Settle()
	if data, err := os.ReadFile(filepath.Join(dir, "Sign in-flow.scenra")); err != nil || string(data) != "flow\n" {
		t.Fatalf("the design's flow: %q %v", data, err)
	}
	app.Press("id:pair-design")
	app.WaitFor("id:files >> label:Sign in-flow", time.Second)
	if got := shownName(app); got != "Sign in" {
		t.Fatalf("the switch showed %q", got)
	}
	app.Press("id:pair-flow")
	if got := shownName(app); got != "Sign in-flow" {
		t.Fatalf("the switch showed %q", got)
	}
	// A component goes in the components folder.
	create("new-component", "tab-bar")
	if data, err := os.ReadFile(filepath.Join(dir, "components", "tab-bar.scenra")); err != nil || string(data) != "component\n" {
		t.Fatalf("the component: %q %v", data, err)
	}
	app.Find("id:files >> label:tab-bar")
	// A flow goes in the folder selected.
	app.Press(`id:files >> text:"flows"`)
	create("new-flow", "checkout")
	if data, err := os.ReadFile(filepath.Join(dir, "flows", "checkout-flow.scenra")); err != nil || string(data) != "flow\n" {
		t.Fatalf("the flow: %q %v", data, err)
	}
	app.Find("id:files >> label:checkout-flow")
	if app.Exists("id:pair") {
		t.Fatal("a flow without its design switches")
	}

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
	if data, _ := os.ReadFile(filepath.Join(dir, "flows", "checkout-flow.scenra")); string(data) != "flow direction=right\n" {
		t.Fatalf("the file was not saved:\n%s", data)
	}
}

func TestDesignsDrawTheComponentsTheyUse(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra":                "design\nbox class=\"flex-1\"\n  text \"Welcome\"\n  use tab-bar class=\"mt-auto\"\n  use gone\nend\n",
		"components/tab-bar.scenra":  "component\nbox class=\"flex-row\"\n  use nav/item\n  text \"Orders\"\nend\n",
		"components/nav/item.scenra": "component\ntext \"Home tab\"\n",
		"components/loop.scenra":     "component\nuse loop\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	app.WaitFor("design >> text:Home tab", 5*time.Second)
	app.Find("id:element-0.1 >> text:Orders")
	// A component's elements select the use drawing them.
	app.Press("design >> text:Orders")
	if got := e.cur().element.GetUntracked(); got != "0.1" {
		t.Fatalf("pressing inside the component selected %q", got)
	}
	// Editing a component redraws the designs using it.
	os.WriteFile(filepath.Join(dir, "components", "nav", "item.scenra"), []byte("component\ntext \"Menu tab\"\n"), 0o644)
	app.WaitFor("design >> text:Menu tab", 5*time.Second)
	// A component opens on its own, at its device's width.
	app.Press("id:files >> label:tab-bar")
	app.WaitFor("design >> text:Orders", time.Second)
	if w := app.Find("id:design").Frame.Size.Width; w != 390 {
		t.Fatalf("the component is %v wide", w)
	}
	app.WaitFor("id:files >> label:loop", time.Second)
	app.Press("id:files >> label:loop")
	app.WaitFor("id:element-0", time.Second)
}

func TestADesignIsDrawnAtTheSizeItSets(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{"wide.scenra": "design width=1440 height=900\ntext \"Wide\"\n"})
	app, e := startEditor(t)
	openPath(t, app, e, dir)
	app.Press("id:files >> label:wide")
	app.WaitFor("design >> text:Wide", time.Second)
	if size := app.Find("id:design").Frame.Size; size.Width != 1440 || size.Height != 900 {
		t.Fatalf("the design is %v", size)
	}
	// An agent resizing the file redraws it at its new size.
	os.WriteFile(filepath.Join(dir, "wide.scenra"), []byte("design width=600 height=400\ntext \"Wide\"\n"), 0o644)
	app.WaitUntil(5*time.Second, func() bool { return app.Find("id:design").Frame.Size.Width == 600 })
}

func TestTheTreeKeepsAFlowByItsDesign(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"sign-in.scenra": "design\n", "sign-in-flow.scenra": "flow\n", "sign-in-error.scenra": "design\n", "about.scenra": "design\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, dir)
	app.WaitFor("id:files >> label:sign-in-error", time.Second)
	if got := treeNames(app); !slices.Equal(got, []string{"about", "sign-in", "sign-in-flow", "sign-in-error"}) {
		t.Fatalf("the tree %v", got)
	}
}

func TestShowingAnotherFileSavesTheOneShown(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{"plan.scenra": "flow\nnode a \"Research\"\n", "home.scenra": "design\ntext \"Welcome\"\n"})
	other := writeFlow(t, "other.scenra", "flow\n")
	plan := filepath.Join(dir, "plan.scenra")
	app, e := startEditor(t, nsgotest.PendingAlerts())
	relabel := func(label string) {
		app.Post(func() {
			d := e.shown()
			f := d.doc.GetUntracked().Clone()
			f.Flow.Nodes[0].Label = label
			d.change(f, "")
		})
		app.Settle()
		app.Find("id:file-edited")
	}
	saved := func(label string) {
		t.Helper()
		if data, _ := os.ReadFile(plan); !strings.Contains(string(data), label) {
			t.Fatalf("the file was not saved:\n%s", data)
		}
		if len(app.Alerts()) != 0 {
			t.Fatalf("leaving the file asked: %+v", app.Alerts())
		}
	}

	// Choosing another file saves the one shown.
	openPath(t, app, e, plan)
	relabel("Edited plan")
	app.WaitFor("id:files >> label:home", time.Second)
	app.Press("id:files >> label:home")
	app.WaitFor("design >> text:Welcome", time.Second)
	saved("Edited plan")

	// So does opening another folder.
	app.Press("id:files >> label:plan")
	app.WaitFor("flow >> label:Edited plan", time.Second)
	relabel("Moved on")
	openPath(t, app, e, other)
	saved("Moved on")
	if got := shownName(app); got != "other" {
		t.Fatalf("the header names %q", got)
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

func TestATabBarMarksTheScreenShown(t *testing.T) {
	item := func(name string) string {
		return "  box class=\"group\" tap=\"open " + name + "\"\n    text \"" + name + " tab\" class=\"text-zinc-400 group-selected:text-amber-600\"\n  end\n"
	}
	dir := folderOfFiles(t, map[string]string{
		"menu.scenra":               "design\nuse tab-bar\n",
		"orders.scenra":             "design\nuse tab-bar\n",
		"components/tab-bar.scenra": "component\nbox class=\"flex-row\"\n" + item("menu") + item("orders") + "end\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "menu.scenra"))
	app.WaitFor("design >> text:orders tab", 5*time.Second)
	if menu, orders := app.Find("design >> text:menu tab").TextStyle.Color, app.Find("design >> text:orders tab").TextStyle.Color; menu == orders {
		t.Fatalf("the tab of the screen shown is %v as the others", menu)
	}
}

func TestExamplesDrawEveryClassAndDrawing(t *testing.T) {
	files, _, _ := scan("../examples", nil)
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
	if app.Exists("id:header") || app.Exists("id:composer") {
		t.Fatal("the views shown brought the header or the assistant")
	}
	if _, ok := app.Storage().GetItem(keyFolder); ok {
		t.Fatal("an editor that remembers nothing kept its folder")
	}
}

func TestDesignsPlayTheirTaps(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra": "design\nbox class=\"flex-1\"\n" +
			"  box class=\"flex-row\"\n" +
			"    box tap=\"tab active\"\n      text \"Active tab\"\n    end\n" +
			"    box tap=\"tab past\"\n      text \"Past tab\"\n    end\n" +
			"  end\n" +
			"  box panel=active\n    text \"Brewing now\"\n  end\n" +
			"  box panel=past\n    text \"Yesterday\"\n  end\n" +
			"  box tap=\"open orders/detail\"\n    text \"Open detail\"\n  end\n" +
			"  text \"Nowhere\" tap=\"open gone\"\n" +
			"  use tab-bar\n" +
			"end\n",
		"orders/detail.scenra":      "design\nbox class=\"flex-1\"\n  text \"Back\" tap=close\n  use tab-bar\nend\n",
		"components/tab-bar.scenra": "component\nbox class=\"flex-row\"\n  box tap=\"open home\"\n    text \"Home item\"\n  end\nend\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	app.WaitFor("design >> text:Brewing now", 5*time.Second)
	// The first panel shows, and a tab shows its own in its place, and
	// the box whose tap shows what shows is selected.
	if app.Exists("design >> text:Yesterday") {
		t.Fatal("a second panel shows")
	}
	selected := func(id string) bool { return app.Find("id:" + id).Props.Selected }
	if !selected("element-0.0.0") || selected("element-0.0.1") {
		t.Fatal("the first tab is not the selected one")
	}
	app.Press("design >> text:Past tab")
	app.WaitFor("design >> text:Yesterday", time.Second)
	app.WaitForGone("design >> text:Brewing now", time.Second)
	if !selected("element-0.0.1") || selected("element-0.0.0") {
		t.Fatal("the tab shown is not the selected one")
	}
	if got := e.cur().element.GetUntracked(); got != "" || app.Exists("id:comment-card") {
		t.Fatalf("playing a tap selected %q", got)
	}
	// A tap opens another design, whose close returns to the one that
	// opened it; the component's tap leading to the design shown is
	// selected.
	app.Press("design >> text:Open detail")
	app.WaitFor("design >> text:Back", time.Second)
	if got := shownName(app); got != "detail" {
		t.Fatalf("the tap opened %q", got)
	}
	app.Press("design >> text:Back")
	app.WaitFor("design >> text:Open detail", time.Second)
	if got := shownName(app); got != "home" {
		t.Fatalf("closing showed %q", got)
	}
	if !selected("element-0.5:0.0") {
		t.Fatal("the tab bar's item for the design shown is not selected")
	}
	// Coming back shows the first tab again, and a close with nothing to
	// return to stays.
	app.Find("design >> text:Brewing now")
	app.Press("design >> text:Nowhere")
	if got := shownName(app); got != "home" {
		t.Fatalf("a tap to no design showed %q", got)
	}
	// What leads nowhere, the agent hears of.
	view := ""
	app.Post(func() { view = e.viewOf() })
	app.Settle()
	if !strings.Contains(view, `home.scenra: tap "open gone": there is no gone.scenra`) {
		t.Fatalf("the view tells nothing of the tap to no design:\n%s", view)
	}
}

func TestTheCardShowsWhereThePointerRests(t *testing.T) {
	shop := writeFlow(t, "shop.scenra", "design\nbox class=\"flex-1 gap-4 p-6\"\n  text \"Welcome\" class=\"text-2xl\"\n"+
		"  box class=\"h-12\" tap=\"open menu\"\n    text \"Menu\"\n  end\nend\n")
	app, e := startEditor(t)
	openPath(t, app, e, shop)
	app.WaitFor("design >> text:Menu", time.Second)
	pane, design := app.Find("id:design-pointer").Frame.Size, app.Find("id:design").Frame.Size
	zoom := min(1, (pane.Width-2*designFitMargin)/design.Width, (pane.Height-2*designFitMargin)/design.Height)
	at := func(path string) core.Point {
		v := app.Find("id:element-" + path)
		x, y := v.Frame.Size.Width/2, v.Frame.Size.Height/2
		for ; v.Props.TestID != "design"; v = v.Parent {
			x, y = x+v.Frame.Origin.X, y+v.Frame.Origin.Y
		}
		return core.Point{X: (pane.Width-design.Width*zoom)/2 + x*zoom, Y: (pane.Height-design.Height*zoom)/2 + y*zoom}
	}
	// Resting on an element shows its card, even on one whose tap
	// pressing plays.
	app.HoverAt("id:design-pointer", at("0.1.0"))
	if app.Exists("id:comment-card") {
		t.Fatal("the card shows before the pointer rests")
	}
	app.Advance(time.Second)
	app.WaitFor("id:comment-card", time.Second)
	app.Find("id:comment-tap >> text:Opens menu")
	if got := e.cur().element.GetUntracked(); got != "" {
		t.Fatalf("resting selected %q", got)
	}
	// A note saves with the element, and keeps the card while it is
	// typed.
	app.Type("id:note-input", "Lists today's drinks")
	app.HoverOut()
	app.Advance(time.Second)
	app.Find("id:comment-card")
	app.Press("id:comment-save")
	app.WaitFor("id:note-pin-0.1.0", time.Second)
	if got := e.shown().doc.GetUntracked().Format(); !strings.Contains(got, `text "Menu" note="Lists today's drinks"`) {
		t.Fatalf("the note was not saved:\n%s", got)
	}
	// Once saved, the card goes when the pointer leaves.
	app.HoverAt("id:design-pointer", at("0.0"))
	app.HoverOut()
	app.Advance(time.Second)
	app.WaitForGone("id:comment-card", time.Second)
	view := ""
	app.Post(func() { view = e.viewOf() })
	app.Settle()
	if !strings.Contains(view, `shop.scenra, element 0.1.0: "Lists today's drinks"`) {
		t.Fatalf("the view tells nothing of the note:\n%s", view)
	}
	app.HoverAt("id:design-pointer", at("0.1.0"))
	app.Advance(time.Second)
	app.WaitFor("id:comment-card", time.Second)
	app.Press("id:note-remove")
	app.WaitForGone("id:note-pin-0.1.0", time.Second)
}
