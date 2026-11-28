package scenraview

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestThumbnailDrawsADesignWithoutTools(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"components/badge.scenra": "component\ntext \"Badge text\"\n",
		"home.scenra":             "design device=phone\nuse badge\ntext \"Welcome\"\n",
	})
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Thumbnail{Path: filepath.Join(dir, "home.scenra"), Size: core.Size{Width: 256, Height: 256}, TestID: "thumbnail"}
	})
	app.WaitFor("id:thumbnail >> id:thumbnail-still >> text:Welcome", 5*time.Second)
	app.WaitFor("id:thumbnail >> id:thumbnail-still >> text:Badge text", 5*time.Second)
	if app.Exists("id:preview-zoom-in") {
		t.Fatal("thumbnail shows the preview's tools")
	}
}

func TestThumbnailDrawsNothingForBrokenEmptyOrMissingFiles(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"broken.scenra":      "design\n  oops\n",
		"flows/empty.scenra": "flow\n",
	})
	for _, name := range []string{"broken.scenra", "flows/empty.scenra", "missing.scenra"} {
		app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
			return Thumbnail{Path: filepath.Join(dir, name), Size: core.Size{Width: 128, Height: 128}, TestID: "thumbnail"}
		})
		if len(app.QueryMounted("id:thumbnail")) == 0 {
			t.Fatalf("%s: no thumbnail", name)
		}
		if app.Exists("id:thumbnail-still") || app.Exists("id:file-broken") {
			t.Fatalf("%s: thumbnail draws something", name)
		}
	}
}
