package scenraview

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestPreviewDrawsADesignFileWithProjectComponents(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"components/badge.scenra": "component\ntext \"Badge text\"\n",
		"home.scenra":             "design device=phone\nuse badge\ntext \"Welcome\"\n",
	})
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Preview{Path: filepath.Join(dir, "home.scenra"), TestID: "preview"}
	})
	app.WaitFor("id:preview >> id:preview-still >> text:Welcome", 5*time.Second)
	app.WaitFor("id:preview >> id:preview-still >> text:Badge text", 5*time.Second)
	app.Find("id:preview >> id:preview-zoom-in")
	app.Find("id:preview >> id:preview-zoom-out")
	app.Find("id:preview >> id:preview-fit")
}

func TestPreviewDrawsAFlowFile(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"flows/checkout.scenra": "flow\nnode start \"Start\" kind=start\nnode done \"Done\" kind=end\nstart -> done \"Pay\"\n",
	})
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Preview{Path: filepath.Join(dir, "flows", "checkout.scenra"), TestID: "preview"}
	})
	app.WaitFor("id:preview >> id:preview-still >> text:Start", 5*time.Second)
	app.WaitFor("id:preview >> id:preview-still >> text:Done", 5*time.Second)
}

func TestPreviewDegradesGracefullyWithoutProjectAccess(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"lone.scenra": "design device=phone\nuse missing-component\ntext \"Standalone\"\n",
	})
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Preview{Path: filepath.Join(dir, "lone.scenra"), TestID: "preview"}
	})
	app.WaitFor("id:preview >> id:preview-still >> text:Standalone", 5*time.Second)
}

func TestPreviewShowsErrorForBrokenOrMissingFiles(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"broken.scenra": "design\n  oops\n",
	})
	broken := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Preview{Path: filepath.Join(dir, "broken.scenra"), TestID: "preview"}
	})
	broken.WaitFor("id:preview >> id:file-broken", 5*time.Second)
	broken.Find("id:preview >> id:file-problem")

	missing := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return Preview{Path: filepath.Join(dir, "missing.scenra"), TestID: "preview"}
	})
	missing.WaitFor("id:preview >> id:file-broken", 5*time.Second)
}
