package scenraview

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
)

func TestUsesPickVariantsAndStates(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra": "design\nbox\n  use tab label=\"Plain\"\n  use tab variant=selected label=\"Picked\"\n  use tab state=selected label=\"Stated\"\nend\n",
		"components/tab.scenra": "component backdrop=\"bg-zinc-100 > bg-black/5\"\nbox name=tab class=\"group selected:bg-zinc-200\"\n" +
			"  text \"Tab\" name=label class=\"text-zinc-400 group-selected:text-amber-600\"\nend\nvariant selected state=selected\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	app.WaitFor("design >> text:Stated", 5*time.Second)
	color := func(text string) core.Color { return app.Find("design >> text:" + text).TextStyle.Color }
	if color("Picked") == color("Plain") || color("Stated") != color("Picked") {
		t.Fatalf("the selected tabs are %v and %v, the plain one %v", color("Picked"), color("Stated"), color("Plain"))
	}
	// The component's file draws its backdrop's layers, the variant as
	// selected as the uses.
	openComponent(t, app, e, dir, "tab")
	app.WaitFor("id:variant-selected", time.Second)
	if n := len(app.Query("id:backdrop-layer")); n != 2 {
		t.Fatalf("%d upper layers drawn", n)
	}
}

// window is a desktop design with a sidebar opening the views that fill
// its content, and those views.
var window = map[string]string{
	"main.scenra": "design device=desktop\nbox class=\"flex-1 flex-row\"\n" +
		"  box class=\"w-56 shrink-0\"\n" +
		"    box class=\"group\" tap=\"open repos\"\n      text \"Repos entry\" class=\"text-zinc-400 group-selected:text-amber-600\"\n    end\n" +
		"    box class=\"group\" tap=\"open issues\"\n      text \"Issues entry\" class=\"text-zinc-400 group-selected:text-amber-600\"\n    end\n" +
		"  end\n" +
		"  box class=\"flex-1\" outlet=content\n    text \"New task\"\n  end\n" +
		"end\n",
	"views/repos.scenra":  "design list in=\"main#content\"\nbox class=\"flex-1\"\n  text \"Repo list\"\n  text \"Back\" tap=close\n  text \"Detail\" tap=\"open #detail\"\nend\n\ndesign detail\nbox class=\"flex-1\"\n  text \"Repo detail\"\nend\n",
	"views/issues.scenra": "design in=\"main#content\"\nbox class=\"flex-1\"\n  text \"Issue list\"\nend\n",
}

func TestOutletsPlayTheDesignsFillingThem(t *testing.T) {
	dir := folderOfFiles(t, window)
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "main.scenra"))
	app.WaitFor("design >> text:New task", 5*time.Second)
	color := func(text string) core.Color { return app.Find("design >> text:" + text).TextStyle.Color }
	plain := color("Issues entry")
	app.Press("design >> text:Issues entry")
	app.WaitFor("design >> text:Issue list", time.Second)
	app.WaitForGone("design >> text:New task", time.Second)
	// The sidebar stays, marking the design shown.
	if color("Issues entry") == plain || color("Repos entry") != plain {
		t.Fatal("the entry of the design shown is not the selected one")
	}
	app.Press("design >> text:Repos entry")
	app.WaitFor("design >> text:Repo list", time.Second)
	app.WaitForGone("design >> text:Issue list", time.Second)
	if shownName(app) != "main" {
		t.Fatalf("%s shows", shownName(app))
	}
	// A design of the file filling the outlet fills it too, and closing
	// returns through what it showed, to its own elements.
	app.Press("design >> text:Detail")
	app.WaitFor("design >> text:Repo detail", time.Second)
	app.Post(func() {})
	if !e.shown().doc.GetUntracked().Several() && shownName(app) != "main" {
		t.Fatal("the outlet left the window")
	}
}

func TestATapOpensAViewByItsName(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra":        "design\ntext \"About entry\" tap=\"open about\"\n",
		"views/about.scenra": "design\ntext \"About page\"\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	app.Press("design >> text:About entry")
	app.WaitFor("design >> text:About page", time.Second)
	if got := e.rel(e.cur().path); got != "views/about.scenra" {
		t.Fatalf("the tap opened %q", got)
	}
}

func TestClosingAnOutletReturnsToWhatItShowed(t *testing.T) {
	dir := folderOfFiles(t, window)
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "main.scenra"))
	app.WaitFor("design >> text:New task", 5*time.Second)
	app.Press("design >> text:Issues entry")
	app.WaitFor("design >> text:Issue list", time.Second)
	app.Press("design >> text:Repos entry")
	app.WaitFor("design >> text:Repo list", time.Second)
	app.Press("design >> text:Back")
	app.WaitFor("design >> text:Issue list", time.Second)
	app.Press("design >> text:Repos entry")
	app.WaitFor("design >> text:Repo list", time.Second)
	app.Press("design >> text:Back")
	app.WaitFor("design >> text:Issue list", time.Second)
}

func TestADesignFillingAnOutletTakesItsSize(t *testing.T) {
	dir := folderOfFiles(t, window)
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "views", "issues.scenra"))
	app.WaitFor("design >> text:Issue list", 5*time.Second)
	// The desktop's 1200 by 760 inside its border, less the sidebar, is
	// the design's, inside its own border.
	app.WaitUntil(time.Second, func() bool {
		f := app.Find("id:element-0/0").Frame.Size
		return f.Width == 1198-224-2 && f.Height == 758-2
	})
}

func TestADesignFillingAnOutletNotLaidOutTakesItsDevicesSize(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"main.scenra":    "design device=desktop\nbox class=\"flex-1\"\n  box class=\"hidden\" outlet=content\n  end\nend\n",
		"views/a.scenra": "design in=\"main#content\"\nbox class=\"flex-1\"\n  text \"Alpha\"\nend\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "views", "a.scenra"))
	app.WaitFor("design >> text:Alpha", 5*time.Second)
	if w := app.Find("id:frame-0").Frame.Size.Width; w != 390 {
		t.Fatalf("the design is %v wide", w)
	}
}

func TestTheServerTellsHowDesignsLayOut(t *testing.T) {
	files := map[string]string{
		"form.scenra": "design\nbox class=\"flex-1 items-center gap-2\"\n" +
			"  box class=\"flex-col\"\n    box class=\"h-10 w-full bg-zinc-100\"\n      text \"Field\"\n    end\n  end\n" +
			"  box class=\"h-6 w-20 overflow-hidden\"\n    text \"A sentence long enough to wrap onto lines the box hides\"\n  end\n" +
			"  box class=\"h-10 w-[600px] bg-rose-500\"\n  end\n" +
			"  use field\n" +
			"end\n",
		"components/field.scenra": "component width=fill\nbox class=\"h-10 border border-zinc-200\"\n  text \"Email\"\nend\n",
		// A column a later class stretches, and a row whose siblings
		// take the rest, fill all they can.
		"padded.scenra": "design\nbox class=\"flex-1 items-center\"\n" +
			"  box class=\"w-full flex-row items-center px-3 flex-col items-stretch\"\n    box class=\"flex-row\"\n      box class=\"h-10 w-full bg-zinc-100\"\n      end\n    end\n  end\n" +
			"  box class=\"w-full flex-row gap-2 px-3\"\n    box class=\"flex-col\"\n      text \"Name\"\n      box class=\"h-10 w-full bg-zinc-100\"\n      end\n    end\n" +
			"    box class=\"h-10 flex-1 bg-zinc-200\"\n    end\n  end\nend\n",
	}
	for name, text := range window {
		files[name] = text
	}
	dir := folderOfFiles(t, files)
	app, e := startEditor(t)
	openPath(t, app, e, dir)
	app.WaitUntil(5*time.Second, func() bool { return len(core.Untracked(e.cx, e.designs.Get)) == 5 })
	wait := func(run func(done func(string))) string {
		var out string
		app.Post(func() { run(func(text string) { out = text }) })
		app.WaitUntil(5*time.Second, func() bool { return out != "" })
		return out
	}
	view := wait(func(done func(string)) { e.viewLaidOut(NotesAll, func(string) {}, done) })
	for _, want := range []string{
		"form.scenra: w-full of 0.0.0 box fills only 0.0 box",
		"content width of 0; give 0.0 w-full",
		`form.scenra: text 0.1.0 text "A sentence long enough to wrap onto line…" is cut off by 0.1 box`,
		"form.scenra: 0.2 box, at -105,",
		"form.scenra: 0.3 use field is",
		"under half the 390 components/field.scenra shows at",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("the view tells nothing of %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "main.scenra:") || strings.Contains(view, "issues.scenra: 0") || strings.Contains(view, "padded.scenra") {
		t.Errorf("the view tells of problems the window has not:\n%s", view)
	}
	layout := wait(func(done func(string)) {
		e.layoutOf("views/issues", "", "", "", -1, func(text string, failed bool) { done(text) })
	})
	if !strings.Contains(layout, "views/issues.scenra laid out 974×758 points") || !strings.Contains(layout, "\n  0.0 text \"Issue list\" at 0,0 ") {
		t.Fatalf("the layout:\n%s", layout)
	} // One element and what it holds tell, down to a depth.
	layout = wait(func(done func(string)) {
		e.layoutOf("form", "", "", "0.0", 1, func(text string, failed bool) { done(text) })
	})
	if !strings.Contains(layout, "Only 0.0 and what it holds:\n0.0 box at ") || !strings.Contains(layout, "\n  0.0.0 box at ") ||
		!strings.Contains(layout, "  … 1 more elements inside; ask for 0.0.0") || strings.Contains(layout, "0.1 box") {
		t.Fatalf("the layout of 0.0:\n%s", layout)
	}
	var failed bool
	wait(func(done func(string)) {
		e.layoutOf("form", "", "", "9", -1, func(text string, f bool) { failed = f; done(text) })
	})
	if !failed {
		t.Fatal("an element that is not there lays out")
	}
}

// settings is a window a main window opens, whose categories fill its
// content and close it from there.
var settings = map[string]string{
	"main.scenra": "design device=desktop\nbox class=\"flex-1\"\n  text \"Main home\"\n  text \"Settings entry\" tap=\"open settings\"\nend\n",
	"settings.scenra": "design general device=desktop\nbox class=\"flex-1 flex-row\"\n" +
		"  box class=\"w-40\"\n    text \"Agents entry\" tap=\"open #agents\"\n  end\n" +
		"  box class=\"flex-1\" outlet=content\n    text \"General page\"\n  end\nend\n\n" +
		"design agents in=\"#general/content\"\nbox class=\"flex-1\"\n  text \"Agents page\"\n" +
		"  text \"Close page\" tap=close\n  text \"Close window\" tap=\"close window\"\n" +
		"  text \"Close settings\" tap=\"close #general\"\n  text \"Close main\" tap=\"close main\"\nend\n",
}

func TestClosingTheWindowAPageShowsIn(t *testing.T) {
	dir := folderOfFiles(t, settings)
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "main.scenra"))
	app.WaitFor("design >> text:Main home", 5*time.Second)
	toAgents := func() {
		t.Helper()
		app.Press("design >> text:Settings entry")
		app.WaitFor("id:element-0/0.1.0", time.Second)
		app.Press("design >> text:Agents entry")
		app.WaitFor("id:element-0/0.1|0.0", time.Second)
	}
	// A close returns the outlet to what it showed.
	toAgents()
	app.Press("id:element-0/0.1|0.1")
	app.WaitForGone("id:element-0/0.1|0.0", time.Second)
	app.WaitFor("id:element-0/0.1.0", time.Second)
	if shownName(app) != "settings" {
		t.Fatalf("%s shows", shownName(app))
	}
	// Closing the window, or the design it is, returns to what opened it,
	// and it opens again showing its outlet's own elements; a design the
	// page does not show in closes nothing.
	for _, tap := range []string{"2", "3"} {
		app.Press("design >> text:Agents entry")
		app.WaitFor("id:element-0/0.1|0.0", time.Second)
		app.Press("id:element-0/0.1|0.4")
		app.Post(func() {})
		app.WaitFor("id:element-0/0.1|0.0", time.Second)
		app.Press("id:element-0/0.1|0." + tap)
		app.WaitFor("design >> text:Main home", time.Second)
		app.Press("design >> text:Settings entry")
		app.WaitFor("id:element-0/0.1.0", time.Second)
	}
	// What no design opened closes nothing.
	app.Press("design >> text:Agents entry")
	app.WaitFor("id:element-0/0.1|0.0", time.Second)
	app.Post(func() { e.trail = nil })
	app.Press("id:element-0/0.1|0.2")
	app.Post(func() {})
	app.WaitFor("id:element-0/0.1|0.0", time.Second)
	view := core.Untracked(e.cx, e.viewOf)
	if !strings.Contains(view, `settings.scenra: tap "close main": close names a design this one does not show in; it shows in settings#agents, settings#general`) ||
		strings.Contains(view, "close window") || strings.Contains(view, "close #general") {
		t.Fatalf("the view:\n%s", view)
	}
}

func TestSideInsetsFollowTailwind(t *testing.T) {
	for class, want := range map[string]core.Points{
		"px-3 py-2 border-b border-border": 24, "p-4 px-2 border": 18, "pl-2.5 pr-[13px] border-x-2": 27,
		"border-l border-zinc-200 hover:px-8": 1, "p-px border-0": 2, "border-y-4 pt-3": 0,
	} {
		if got := sideInsets(class); got != want {
			t.Errorf("%q: %v, want %v", class, got, want)
		}
	}
}
