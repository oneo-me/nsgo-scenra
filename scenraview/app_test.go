package scenraview

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
	// What can be tapped glows a moment, which a step would otherwise
	// wait out with the clock.
	app.Advance(2 * time.Second)
}

// startEditor starts an app of the Window, whose assistant's agent is
// driver, and returns its editor.
func startEditor(t *testing.T, opts ...nsgotest.Option) (*nsgotest.App, *Editor) {
	app, e, _ := startAssistant(t, opts...)
	return app, e
}

// shownName is the name the header gives the file shown.
func shownName(app *nsgotest.App) string { return app.Find("id:header >> id:file-name").Text }

// openComponent opens the component named name of the folder dir, as
// making it or an agent's turn does.
func openComponent(t *testing.T, app *nsgotest.App, e *Editor, dir, name string) {
	t.Helper()
	app.Post(func() { e.open(filepath.Join(dir, "components", filepath.FromSlash(name)+".scenra")) })
	app.WaitFor("id:design", time.Second)
}

// fileNames are the names of the sidebar's files, in order.
func fileNames(app *nsgotest.App) []string {
	var names []string
	for _, v := range app.Query("id:files >> id:file") {
		names = append(names, v.Props.Label)
	}
	return names
}

func TestShowsTheFolderInGroups(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"plan.scenra":             "flow\nnode a \"Research\" note=\"Read the docs\"\na -> b\n",
		"home.scenra":             "design\ntext \"Welcome\"\n",
		"broken.scenra":           "box\n",
		"cart.scenra":             "design name=\"Shop/Shopping cart\"\ntext \"Cart\"\n",
		"cart-flow.scenra":        "flow\nnode a \"Pay\"\n",
		"deep.scenra":             "design name=\"A/B/Deep\"\n",
		"screens/old.scenra":      "design\n",
		".hidden/secret.scenra":   "flow\n",
		"node_modules/pkg.scenra": "flow\n",
		"notes.txt":               "design\n",
		"empty/nothing-here.md":   "",
	})
	app, e := startEditor(t)
	app.Find("id:sidebar-open-folder")
	openPath(t, app, e, dir)
	app.WaitFor("id:files >> label:plan", time.Second)
	if got := app.Find("id:header >> id:folder-name").Text; got != filepath.Base(dir) {
		t.Fatalf("the header names the folder %q", got)
	}
	if got := fileNames(app); !slices.Equal(got, []string{"A/B/Deep", "broken", "home", "plan", "Shopping cart", "cart-flow"}) {
		t.Fatalf("the sidebar lists %v", got)
	}
	// A name's group holds the file, and its design's holds a flow.
	app.Find("id:group-Shop >> label:Shopping cart")
	app.Find("id:group-Shop >> label:cart-flow")
	// Files in folders are not the project's.
	for _, name := range []string{"old", "screens", "secret", "pkg", "notes", "empty", "cart"} {
		if app.Exists("id:files >> label:" + name) {
			t.Errorf("the sidebar shows %s", name)
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
	if got := shownName(app); got != "Shop/Shopping cart" {
		t.Fatalf("the header names %q", got)
	}
	app.Press("id:files >> label:home")
	app.WaitFor("design >> text:Welcome", time.Second)
	// A file another program makes shows in the sidebar.
	os.WriteFile(filepath.Join(dir, "later.scenra"), []byte("flow\n"), 0o644)
	app.WaitFor("id:files >> label:later", 5*time.Second)

	// The folder and the file shown open again at the next start.
	stored, _ := app.Storage().GetItem(keyFile)
	if stored != filepath.Join(dir, "home.scenra") {
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
	pane, design := app.Find("id:design-camera-pointer").Frame.Size, app.Find("id:design").Frame.Size
	zoom := min(1, (pane.Width-2*designFitMargin)/design.Width, (pane.Height-2*designFitMargin)/design.Height)
	if zoom >= 1 {
		t.Fatalf("the design shows unscaled at %v in %v", design, pane)
	}
	at := func(path string) core.Point {
		v := app.Find("id:element-0/" + path)
		x, y := v.Frame.Size.Width/2, v.Frame.Size.Height/2
		for ; v.Props.TestID != "design"; v = v.Parent {
			x, y = x+v.Frame.Origin.X, y+v.Frame.Origin.Y
		}
		return core.Point{X: (pane.Width-design.Width*zoom)/2 + x*zoom, Y: (pane.Height-design.Height*zoom)/2 + y*zoom}
	}
	app.HoverAt("id:design-camera-pointer", at("0.1.0"))
	app.WaitFor("id:design-hovered", time.Second)
	app.PressAt("id:design-camera-pointer", at("0.1.0"))
	if got := e.cur().element.GetUntracked(); got != "0/0.1.0" {
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
	if got := e.cur().element.GetUntracked(); got != "0/0.1.0" || !app.Exists("id:comment-card") {
		t.Fatalf("typing in the card selected %q", got)
	}
	// Pressing the card's blank space leaves the canvas under it alone.
	app.PressAt("id:comment-card", core.Point{X: 4, Y: 4})
	if got := e.cur().element.GetUntracked(); got != "0/0.1.0" || !app.Exists("id:comment-card") {
		t.Fatalf("pressing the card selected %q", got)
	}
	app.HoverAt("id:design-camera-pointer", at("0.0"))
	app.PressAt("id:design-camera-pointer", at("0.0"))
	if got := e.cur().element.GetUntracked(); got != "0/0.0" {
		t.Fatalf("pressing the title selected %q", got)
	}
	app.HoverOut()
	if app.Exists("id:design-hovered") {
		t.Fatal("the outline stayed after the pointer left")
	}
	app.PressAt("id:design-camera-pointer", core.Point{X: 4, Y: 4})
	if got := e.cur().element.GetUntracked(); got != "" {
		t.Fatalf("pressing around the design left %q selected", got)
	}
}

func TestHidingThePanelsLeavesTheWindowToTheWorkspace(t *testing.T) {
	app, e := startEditor(t)
	openPath(t, app, e, writeFlow(t, "home.scenra", "design\nview\n"))
	app.WaitFor("id:files >> label:home", time.Second)
	waitComposer(app)
	width := app.Find("id:header").Frame.Size.Width
	app.Post(e.TogglePanels)
	app.Settle()
	if app.Exists("id:files") || app.Exists("id:composer") {
		t.Fatal("the hidden panels still show")
	}
	if got := app.Find("id:header").Frame.Size.Width; got != 1280 {
		t.Fatalf("the header is %v wide, from %v", got, width)
	}
	// Making a file shows the sidebar, holding its dialog.
	app.Post(e.NewDesign)
	app.WaitFor("id:new-file-dialog", time.Second)
	app.Find("id:files >> label:home")
	if !app.Exists("id:composer") {
		t.Fatal("the assistant stays hidden")
	}
}

func TestMakesFilesInTheFolder(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{"old.scenra": "flow\n"})
	app, e := startEditor(t)
	openPath(t, app, e, dir)
	app.WaitFor("id:files >> label:old", time.Second)
	ask := func(item string) {
		app.Post(map[string]func(){"new-design": e.NewDesign, "new-flow": e.NewFlow, "new-component": e.NewComponent}[item])
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
	if app.Exists("id:files >> label:tab-bar") {
		t.Fatal("the sidebar lists the component")
	}
	app.Find("id:sidebar-components")
	// Every file goes at the top of the folder.
	app.Press("id:files >> label:old")
	create("new-flow", "checkout")
	if data, err := os.ReadFile(filepath.Join(dir, "checkout-flow.scenra")); err != nil || string(data) != "flow\n" {
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
		f.Flows[0].Direction = flowfile.Right
		d.change(f, "")
	})
	app.Settle()
	app.Post(e.Save)
	app.Settle()
	if data, _ := os.ReadFile(filepath.Join(dir, "checkout-flow.scenra")); string(data) != "flow direction=right\n" {
		t.Fatalf("the file was not saved:\n%s", data)
	}
}

func TestDesignsDrawTheComponentsTheyUse(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra":                "design\nbox class=\"flex-1\"\n  text \"Welcome\"\n  use tab-bar class=\"mt-auto\"\n  use gone\nend\n",
		"components/tab-bar.scenra":  "component width=fill\nbox class=\"flex-row\"\n  use nav-item\n  text \"Orders\"\nend\n",
		"components/nav-item.scenra": "component\ntext \"Home tab\"\n",
		"components/loop.scenra":     "component\nuse loop\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	app.WaitFor("design >> text:Home tab", 5*time.Second)
	app.Find("id:element-0/0.1 >> text:Orders")
	// A component's elements select the use drawing them.
	app.Press("design >> text:Orders")
	if got := e.cur().element.GetUntracked(); got != "0/0.1" {
		t.Fatalf("pressing inside the component selected %q", got)
	}
	// Editing a component redraws the designs using it.
	os.WriteFile(filepath.Join(dir, "components", "nav-item.scenra"), []byte("component\ntext \"Menu tab\"\n"), 0o644)
	app.WaitFor("design >> text:Menu tab", 5*time.Second)
	// A component opens on its own, at its device's width when it fills
	// the width, and else as large as it needs.
	openComponent(t, app, e, dir, "tab-bar")
	app.WaitFor("design >> text:Orders", time.Second)
	if w := app.Find("id:component-0").Frame.Size.Width; w != 390 {
		t.Fatalf("the component is %v wide", w)
	}
	app.Find("id:component-0 >> id:component-outline")
	openComponent(t, app, e, dir, "nav-item")
	app.WaitFor("design >> text:Menu tab", time.Second)
	if w, text := app.Find("id:component-0").Frame.Size.Width, app.Find("id:element-0/0").Frame.Size.Width; w != text {
		t.Fatalf("the component is %v wide around a text %v wide", w, text)
	}
	openComponent(t, app, e, dir, "loop")
	app.WaitFor("id:element-0/0", time.Second)
}

func TestImagesShowThePicturesOfTheAssetsFolder(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra":            "design\nimage \"Latte\" src=\"assets/latte.png\"\nimage \"Gone\" src=\"assets/gone.png\"\nuse card pic.src=\"./assets/cups/mocha.png\"\n",
		"components/card.scenra": "component\nimage \"Cup\" name=pic\n",
		"assets/latte.png":       "latte",
		"assets/cups/mocha.png":  "mocha",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	picture := func(sel string) string { return string(app.Find(sel).Image.Source.Data) }
	app.WaitUntil(5*time.Second, func() bool { return picture("id:element-0/0") == "latte" })
	if got := picture("id:element-0/2"); got != "mocha" {
		t.Fatalf("the use shows %q", got)
	}
	if app.Find("id:element-0/1").Image.Source.Data != nil {
		t.Fatal("a missing picture shows")
	}
	// Changing a picture redraws the designs showing it.
	os.WriteFile(filepath.Join(dir, "assets", "latte.png"), []byte("latte art"), 0o644)
	app.WaitUntil(5*time.Second, func() bool { return picture("id:element-0/0") == "latte art" })
}

func TestAUseDrawsAsTheElementAtTheTopOfItsComponent(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra": "design\nbox class=\"flex-1 items-center\"\n  use composer class=\"w-full max-w-2xl\"\n  box class=\"h-10 w-full max-w-2xl\"\n  end\n" +
			"  use pair class=\"w-full\"\nend\n",
		"components/composer.scenra": "component\nbox class=\"h-10 w-20 flex-row\"\n  text \"Ask\"\nend\n",
		"components/pair.scenra":     "component\ntext \"One\"\ntext \"Two\"\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	app.WaitFor("id:element-0/0.0 >> text:Ask", 5*time.Second)
	// The use's classes go to the component's box, winning over its own,
	// so it lays out as the box written in its place.
	if use, box := app.Find("id:element-0/0.0").Frame, app.Find("id:element-0/0.1").Frame; use.Size != box.Size || use.Origin.X != box.Origin.X {
		t.Fatalf("the use lays out at %v, the box at %v", use, box)
	}
	// A component of several elements is held by a box of the use's
	// classes.
	if w := app.Find("id:element-0/0.2").Frame.Size.Width; w != app.Find("id:element-0/0.1").Frame.Size.Width {
		t.Fatalf("the box holding the pair is %v wide", w)
	}
	// Its elements still select the use, outlined where it is.
	app.Press("design >> text:Ask")
	if got := e.cur().element.GetUntracked(); got != "0/0.0" {
		t.Fatalf("pressing inside the component selected %q", got)
	}
	app.Settle()
	use, mark := app.Find("id:element-0/0.0").Frame, app.Find("id:design-selected").Frame
	if mark.Size.Width != use.Size.Width+4 || mark.Size.Height != use.Size.Height+4 {
		t.Fatalf("the use is at %v, its outline at %v", use, mark)
	}
}

func TestAUseChangesTheNamedElementsOfItsComponent(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"menu.scenra": "design\nbox class=\"flex-1\"\n" +
			"  use item title=Latte price=\"$4.20\" price.class=\"text-rose-600\" row.tap=\"open detail\"\n" +
			"  use item title=\"Cold brew\" badge/label=Hot nope=x\n" +
			"end\n",
		"detail.scenra":           "design\ntext \"Detail\" tap=close\n",
		"components/item.scenra":  "component\nbox name=row class=\"flex-row\"\n  text Title name=title\n  text \"$0\" name=price\n  use badge name=badge\nend\n",
		"components/badge.scenra": "component\ntext New name=label\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "menu.scenra"))
	app.WaitFor("id:element-0/0.0 >> text:Latte", 5*time.Second)
	app.Find("id:element-0/0.0 >> text:$4.20")
	app.Find("id:element-0/0.0 >> text:New")
	app.Find("id:element-0/0.1 >> text:Cold brew")
	app.Find("id:element-0/0.1 >> text:$0")
	app.Find("id:element-0/0.1 >> text:Hot")
	// A tap a use gives its component plays.
	app.Press("design >> text:Latte")
	app.WaitFor("design >> text:Detail", time.Second)
	app.Press("design >> text:Detail")
	app.WaitFor("design >> text:Cold brew", time.Second)
	// What a use changes that is not there, the agent hears of.
	view := ""
	app.Post(func() { view = e.viewOf() })
	app.Settle()
	if !strings.Contains(view, `menu.scenra: use "item": nope="x" names nothing: no element of components/item.scenra has name=nope`) {
		t.Fatalf("the view tells nothing of the override of no element:\n%s", view)
	}
}

func TestAUseFillsTheSlotsOfItsComponent(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra": "design\n" +
			"use card class=\"m-4\"\n  fill header\n    text \"Latte\"\n  end\n" +
			"  fill\n    use pill\n      fill\n        text \"Hot\"\n      end\n    end\n  end\n  fill footer\n  end\nend\n" +
			"use card\n",
		"components/card.scenra":  "component\nbox class=\"gap-2\"\n  slot header\n    text \"Untitled\"\n  end\n  use frame\n    fill\n      slot\n      end\n    end\n  end\nend\n",
		"components/frame.scenra": "component\nbox class=\"p-2\"\n  slot\n  end\nend\n",
		"components/pill.scenra":  "component\nslot class=\"rounded-full\"\nend\n",
		"components/c.scenra":     "component\nuse d\n",
		"components/d.scenra":     "component\nuse pill\n  fill\n    use c\n  end\nend\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	app.WaitFor("id:element-0/0 >> text:Latte", 5*time.Second)
	app.Find("id:element-0/0 >> text:Hot")
	app.Find("id:element-0/1 >> text:Untitled")
	// What a fill holds is the design's: it selects itself and is
	// outlined where its slot draws it.
	app.Press("design >> text:Hot")
	if got := e.cur().element.GetUntracked(); got != "0/0.1.0.0.0" {
		t.Fatalf("pressing what the fill holds selected %q", got)
	}
	app.Settle()
	text, mark := app.Find("id:element-0/0.1.0.0.0").Frame, app.Find("id:design-selected").Frame
	if mark.Size.Width != text.Size.Width+4 || mark.Size.Height != text.Size.Height+4 {
		t.Fatalf("the text is at %v, its outline at %v", text, mark)
	}
	app.Press("design >> text:Untitled")
	if got := e.cur().element.GetUntracked(); got != "0/1" {
		t.Fatalf("pressing what the slot holds selected %q", got)
	}
	// Fills of no slot and loops through fills, the agent hears of.
	view := ""
	app.Post(func() { view = e.viewOf() })
	app.Settle()
	for _, want := range []string{
		`home.scenra: use "card": the fill of slot footer fills nothing, as components/card.scenra has no such slot; it has slot header, the default slot`,
		`components/c.scenra: use "d": components use each other in a loop, c uses d uses c`,
	} {
		if !strings.Contains(view, want) {
			t.Errorf("the view tells nothing of %q:\n%s", want, view)
		}
	}
	// A component in a loop still opens, and its own file shows its slots.
	openComponent(t, app, e, dir, "c")
	app.WaitFor("id:element-0/0", time.Second)
	openComponent(t, app, e, dir, "card")
	app.WaitFor("id:element-0/0.0 >> text:Untitled", time.Second)
	app.Find("id:element-0/0.1:0.0")
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

func TestTheSidebarKeepsAFlowByItsDesign(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"sign-in.scenra": "design\n", "sign-in-flow.scenra": "flow\n", "sign-in-error.scenra": "design\n", "about.scenra": "design\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, dir)
	app.WaitFor("id:files >> label:sign-in-error", time.Second)
	if got := fileNames(app); !slices.Equal(got, []string{"about", "sign-in", "sign-in-flow", "sign-in-error"}) {
		t.Fatalf("the sidebar lists %v", got)
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
			f.Flows[0].Nodes[0].Label = label
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
		return "  box name=" + name + " class=\"group\"\n    text \"" + name + " tab\" class=\"text-zinc-400 group-selected:text-amber-600\"\n  end\n"
	}
	bar := "use tab-bar menu.tap=\"open menu\" orders.tap=\"open orders\"\n"
	dir := folderOfFiles(t, map[string]string{
		"menu.scenra":               "design\n" + bar,
		"orders.scenra":             "design\n" + bar,
		"components/tab-bar.scenra": "component\nbox class=\"flex-row\"\n" + item("menu") + item("orders") + "end\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "menu.scenra"))
	app.WaitFor("design >> text:orders tab", 5*time.Second)
	if menu, orders := app.Find("design >> text:menu tab").TextStyle.Color, app.Find("design >> text:orders tab").TextStyle.Color; menu == orders {
		t.Fatalf("the tab of the screen shown is %v as the others", menu)
	}
}

func TestTapsAUseGivesShowThePanelsOfItsDesign(t *testing.T) {
	item := func(name string) string {
		return "  box name=" + name + "\n    text \"" + name + " item\"\n  end\n"
	}
	dir := folderOfFiles(t, map[string]string{
		"home.scenra": "design\nbox class=\"flex-row\"\n" +
			"  use nav inbox.tap=\"tab inbox\" archive.tap=\"tab archive\"\n" +
			"  box panel=inbox\n    text \"No new mail\"\n  end\n" +
			"  box panel=archive\n    text \"Nothing archived\"\n  end\n" +
			"end\n",
		"components/nav.scenra": "component\nbox\n" + item("inbox") + item("archive") + "end\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	app.WaitFor("design >> text:No new mail", 5*time.Second)
	selected := func(id string) bool { return app.Find("id:" + id).Props.Selected }
	if !selected("element-0/0.0:0.0") || selected("element-0/0.0:0.1") {
		t.Fatal("the item of the first panel is not the selected one")
	}
	app.Press("design >> text:archive item")
	app.WaitFor("design >> text:Nothing archived", time.Second)
	app.WaitForGone("design >> text:No new mail", time.Second)
	if !selected("element-0/0.0:0.1") || selected("element-0/0.0:0.0") {
		t.Fatal("the item of the panel shown is not the selected one")
	}
}

func TestExamplesDrawEveryClassAndDrawing(t *testing.T) {
	files, parts, _, _, _ := scan("../examples/coffee-shop", nil)
	for _, en := range files {
		f, _ := readFile(filepath.Join("../examples/coffee-shop", en.path))
		self, _ := flowfile.ComponentName(en.path)
		for _, s := range f.Screens {
			if p := componentProblems(s, self, parts); len(p) > 0 {
				t.Errorf("%s: %v", en.path, p)
			}
		}
	}
	files, _, _, _, _ = scan("../examples", nil)
	if len(files) < 2 {
		t.Fatalf("examples %v", files)
	}
	for _, en := range files {
		path := filepath.Join("../examples", en.path)
		f, err := readFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for _, s := range f.Screens {
			if unknown := unknownClasses(s); len(unknown) > 0 {
				t.Errorf("%s: classes that draw nothing: %v", path, unknown)
			}
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
		for _, s := range f.Screens {
			walk(s.UI)
		}
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
			"  box panel=past\n    text \"Yesterday\" tap=close\n  end\n" +
			"  box tap=\"open order-detail\"\n    text \"Open detail\"\n  end\n" +
			"  text \"Nowhere\" tap=\"open gone\"\n" +
			"  use tab-bar\n" +
			"end\n",
		"order-detail.scenra":       "design\nbox class=\"flex-1\"\n  text \"Back\" tap=close\n  use tab-bar\nend\n",
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
	// What can be tapped is marked, in the components used too, and the
	// toolbar shows it again.
	marked := func(want ...string) {
		t.Helper()
		var got []string
		for _, v := range app.Find("id:design-taps").Children {
			got = append(got, strings.TrimPrefix(v.Props.TestID, "design-tap-"))
		}
		if !slices.Equal(got, want) {
			t.Fatalf("marked %v, want %v", got, want)
		}
	}
	marked("0/0.0.0", "0/0.0.1", "0/0.3", "0/0.4", "0/0.5:0.0")
	app.Press("id:design-show-taps")
	selected := func(id string) bool { return app.Find("id:" + id).Props.Selected }
	if !selected("element-0/0.0.0") || selected("element-0/0.0.1") {
		t.Fatal("the first tab is not the selected one")
	}
	app.Press("design >> text:Past tab")
	app.WaitFor("design >> text:Yesterday", time.Second)
	app.WaitForGone("design >> text:Brewing now", time.Second)
	marked("0/0.0.0", "0/0.0.1", "0/0.2.0", "0/0.3", "0/0.4", "0/0.5:0.0")
	if !selected("element-0/0.0.1") || selected("element-0/0.0.0") {
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
	if got := shownName(app); got != "order-detail" {
		t.Fatalf("the tap opened %q", got)
	}
	app.Press("design >> text:Back")
	app.WaitFor("design >> text:Open detail", time.Second)
	if got := shownName(app); got != "home" {
		t.Fatalf("closing showed %q", got)
	}
	if !selected("element-0/0.5:0.0") {
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
	pane, design := app.Find("id:design-camera-pointer").Frame.Size, app.Find("id:design").Frame.Size
	zoom := min(1, (pane.Width-2*designFitMargin)/design.Width, (pane.Height-2*designFitMargin)/design.Height)
	at := func(path string) core.Point {
		v := app.Find("id:element-0/" + path)
		x, y := v.Frame.Size.Width/2, v.Frame.Size.Height/2
		for ; v.Props.TestID != "design"; v = v.Parent {
			x, y = x+v.Frame.Origin.X, y+v.Frame.Origin.Y
		}
		return core.Point{X: (pane.Width-design.Width*zoom)/2 + x*zoom, Y: (pane.Height-design.Height*zoom)/2 + y*zoom}
	}
	// Resting on an element shows its card, even on one whose tap
	// pressing plays.
	app.HoverAt("id:design-camera-pointer", at("0.1.0"))
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
	app.WaitFor("id:note-pin-0/0.1.0", time.Second)
	if got := e.shown().doc.GetUntracked().Format(); !strings.Contains(got, `text "Menu" note="Lists today's drinks"`) {
		t.Fatalf("the note was not saved:\n%s", got)
	}
	// Once saved, the card goes when the pointer leaves.
	app.HoverAt("id:design-camera-pointer", at("0.0"))
	app.HoverOut()
	app.Advance(time.Second)
	app.WaitForGone("id:comment-card", time.Second)
	view := ""
	app.Post(func() { view = e.viewOf() })
	app.Settle()
	if !strings.Contains(view, `shop.scenra, element 0.1.0: "Lists today's drinks"`) {
		t.Fatalf("the view tells nothing of the note:\n%s", view)
	}
	app.HoverAt("id:design-camera-pointer", at("0.1.0"))
	app.Advance(time.Second)
	app.WaitFor("id:comment-card", time.Second)
	app.Press("id:note-remove")
	app.WaitForGone("id:note-pin-0/0.1.0", time.Second)
}

func TestAFileHoldsSeveralDesigns(t *testing.T) {
	auth := writeFlow(t, "auth.scenra", "design sign-in name=\"Sign in\" gap=40\n"+
		"box class=\"flex-1 p-4\"\n  text \"Welcome\"\n  text \"Create one\" tap=\"open #sign-up\"\nend\n\n"+
		"design sign-up name=\"Sign up\"\n"+
		"box class=\"flex-1 p-4\"\n  text \"Join us\" comment=\"Warmer\"\n  text \"Back\" tap=close\nend\n")
	app, e := startEditor(t)
	openPath(t, app, e, auth)
	app.WaitFor("design >> text:Join us", time.Second)
	if got := shownName(app); got != "Sign in" {
		t.Fatalf("the file is called %q", got)
	}
	// The designs lay out side by side the file's gap apart, under their
	// names.
	in, up := app.Find("id:screen-sign-in").Frame, app.Find("id:screen-sign-up").Frame
	if up.Origin.X != in.Origin.X+390+40 || up.Origin.Y != in.Origin.Y || in.Origin.Y != captionRoom {
		t.Fatalf("the designs are at %v and %v", in.Origin, up.Origin)
	}
	app.Find("id:design >> text:Sign up")
	// The harness hit-tests without the canvas's transform, so the second
	// design's pin counts as clipped: look for it in the tree.
	if !strings.Contains(app.Dump(), "id=comment-pin-1/0.0") {
		t.Fatal("the second design's comment has no pin")
	}
	// Pressing selects the element of its design.
	app.Press("design >> text:Join us")
	if got := e.cur().element.GetUntracked(); got != "1/0.0" {
		t.Fatalf("pressing selected %q", got)
	}
	// A tap opening a design of the file moves to it, and a close moves
	// back.
	app.Press("design >> text:Create one")
	app.Settle()
	if got := e.focus.GetUntracked().id; got != "sign-up" || shownName(app) != "Sign in" {
		t.Fatalf("the tap moved to %q", got)
	}
	app.Press("design >> text:Back")
	app.Settle()
	if got := e.focus.GetUntracked().id; got != "sign-in" || len(e.trail) != 0 {
		t.Fatalf("closing moved to %q, the trail %v", got, e.trail)
	}
	// A comment on the second design is set on it.
	app.Post(func() { e.shown().setComment("1/0.1", "Arrow") })
	app.Settle()
	if c := e.cur().doc.GetUntracked().Comments(); len(c) != 2 || c[1] != (flowfile.Comment{Screen: 1, Path: "0.1", Text: "Arrow"}) {
		t.Fatalf("comments %v", c)
	}
}

func TestAFileHoldsSeveralFlows(t *testing.T) {
	pay := writeFlow(t, "pay-flow.scenra", "flow pay name=\"Pay\" gap=200\nnode card \"Charge card\"\ncard -> receipt\n\n"+
		"flow refund name=\"Refund\" direction=right\nnode ask \"Ask why\"\nask -> back\n")
	app, e := startEditor(t)
	openPath(t, app, e, pay)
	app.WaitFor("flow >> label:Ask why", 5*time.Second)
	app.WaitUntil(5*time.Second, func() bool { return e.chart(0).Ready() && e.chart(1).Ready() })
	if !strings.Contains(app.Dump(), `"Refund"`) {
		t.Fatal("the second flow has no caption")
	}
	// pos is where the node with key is in the plane of the flows.
	pos := func(key string) (float32, float32) {
		flow, id, _ := splitNode(key)
		x, y, ok := e.chart(flow).Position(id)
		if !ok {
			t.Fatalf("%s is not laid out", key)
		}
		o := app.FindMounted("id:flow-" + strconv.Itoa(flow)).Frame.Origin
		return x + float32(o.X), y + float32(o.Y)
	}
	// Each flow lays out apart, the way it points.
	cx, cy := pos("0/card")
	rx, ry := pos("0/receipt")
	ax, ay := pos("1/ask")
	bx, by := pos("1/back")
	if cx != rx || ry <= cy || ay != by || bx <= ax {
		t.Fatalf("the flows lay out as %v,%v %v,%v and %v,%v %v,%v", cx, cy, rx, ry, ax, ay, bx, by)
	}
	if ax < rx+200 && ay < ry+200 {
		t.Fatal("the flows are closer than the file's gap")
	}
	// Turning the selected node's flow leaves the other.
	app.Post(func() {
		e.shown().selected.Set("1/ask")
		e.ToggleDirection()
	})
	app.Settle()
	f := e.cur().doc.GetUntracked()
	if f.Flows[0].Direction != "" || f.Flows[1].Direction != flowfile.Down {
		t.Fatalf("turned %q and %q", f.Flows[0].Direction, f.Flows[1].Direction)
	}
}
