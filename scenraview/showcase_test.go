package scenraview

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestShowcaseScalesSmallComponentsToTheLeastArea(t *testing.T) {
	got := showcaseScales([]core.Size{{Width: 390, Height: 600}, {Width: 390, Height: 600}, {Width: 100, Height: 40}, {Width: 390, Height: 150}})
	if got[0] != 1 || got[1] != 1 {
		t.Fatalf("the largest scale %v", got[:2])
	}
	if got[2] != showcaseMaxScale {
		t.Fatalf("a button scales %v, not at most %v", got[2], showcaseMaxScale)
	}
	// A card a little under the least area grows just to reach it.
	if area := 390 * 150 * got[3] * got[3]; math.Abs(area-390*600*showcaseMinShare) > 1 {
		t.Fatalf("the card shows %v points², scaled %v", area, got[3])
	}
	// Alone, a component needs only the least area of all.
	if got := showcaseScales([]core.Size{{Width: 100, Height: 40}}); got[0] <= 1 || math.Abs(100*40*got[0]*got[0]-showcaseMinArea) > 1 {
		t.Fatalf("a lone button scales %v", got[0])
	}
}

func TestTheComponentsShowTogether(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra":                "design\nuse tab-bar\n",
		"components/tab-bar.scenra":  "component name=\"Tab bar\"\nbox class=\"flex-row\"\n  use nav/item\n  text \"Orders\"\nend\n",
		"components/nav/item.scenra": "component width=200\ntext \"Home tab\"\n",
		"components/card.scenra":     "component\nbox\n  slot\n  end\nend\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	app.WaitFor("id:sidebar-components", 5*time.Second)
	app.Press("id:sidebar-components")
	app.WaitFor("id:showcase >> text:Orders", time.Second)
	if got := shownName(app); got != "Components" {
		t.Fatalf("the header names %q", got)
	}
	// Each draws as its own file does, at the size it sets.
	app.Find("id:showcase-tab-bar >> text:Home tab")
	app.Find("id:showcase-card")
	if w := app.Find("id:showcase-drawing-nav/item").Frame.Size.Width; w != 200 {
		t.Fatalf("the component is %v wide", w)
	}
	// A small one shows larger, to the least area the largest gives it.
	app.Advance(time.Second)
	if w := app.Find("id:showcase-nav/item").Frame.Size.Width; w <= 200 {
		t.Fatalf("the small component shows %v wide", w)
	}
	// The tree leaves the components to the showcase.
	if app.Exists("id:files >> label:Tab bar") || app.Exists("id:files >> label:components") {
		t.Fatal("the tree shows the components")
	}
	// They pack apart at the heights they laid out at.
	app.Advance(time.Second)
	ids := []string{"card", "nav/item", "tab-bar"}
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			fa, fb := app.Find("id:showcase-"+a).Frame, app.Find("id:showcase-"+b).Frame
			if fa.Origin.X < fb.Origin.X+fb.Size.Width && fb.Origin.X < fa.Origin.X+fa.Size.Width &&
				fa.Origin.Y < fb.Origin.Y+fb.Size.Height && fb.Origin.Y < fa.Origin.Y+fa.Size.Height {
				t.Fatalf("%s at %v overlaps %s at %v", a, fa, b, fb)
			}
		}
	}
	// Choosing the file shown in the tree shows it again.
	app.Press("id:files >> label:home")
	app.WaitFor("id:design", time.Second)
	if app.Exists("id:showcase") {
		t.Fatal("the components still show")
	}
	// Pressing a component only shows it.
	app.Press("id:sidebar-components")
	app.WaitFor("id:showcase-tab-bar", time.Second)
	app.PressAt("id:showcase-camera-pointer", showcasePoint(app, "tab-bar"))
	app.Advance(time.Second)
	if !e.showcase.GetUntracked() {
		t.Fatal("pressing a component left the showcase")
	}
}

// showcasePoint is the middle of the component named name in the fitted
// showcase, in the camera's view.
func showcasePoint(app *nsgotest.App, name string) core.Point {
	pane, all := app.Find("id:showcase-camera-pointer").Frame.Size, app.Find("id:showcase").Frame.Size
	zoom := min(1, (pane.Width-2*designFitMargin)/all.Width, (pane.Height-2*designFitMargin)/all.Height)
	b := app.Find("id:showcase-" + name).Frame
	x, y := b.Origin.X+b.Size.Width/2, b.Origin.Y+b.Size.Height/2
	return core.Point{X: (pane.Width-all.Width*zoom)/2 + x*zoom, Y: (pane.Height-all.Height*zoom)/2 + y*zoom}
}
