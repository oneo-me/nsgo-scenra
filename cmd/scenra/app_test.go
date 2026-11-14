package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

// TestMain gives the platform the main thread under nsgo test -target.
func TestMain(m *testing.M) { nsgotest.Main(m) }

// model fails every request, as a provider that is set but broken.
func model(t *testing.T) string {
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"message":"no more replies"}}`)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestSettingsSetTheProvider(t *testing.T) {
	app := nsgotest.Start(t, Root, nsgotest.Size(1280, 800))
	app.Type("label:Message", "Hello")
	app.Press("label:Send")
	app.WaitFor("id:settings", time.Second)
	app.Type("id:model", "test-model")
	app.Clear("id:base-url")
	app.Type("id:base-url", model(t))
	app.Press("id:settings-done")
	app.WaitForGone("id:settings", 2*time.Second)
	path := filepath.Join(t.TempDir(), "hello.scenra")
	app.SaveFile = func(core.SaveRequest) string { return path }
	app.Press("label:Send")
	app.WaitFor("text:no more replies (HTTP 500)", 5*time.Second)
	app.Find("id:title-bar >> label:hello.scenra")
}

func TestSettingsTurnOffTheStatistics(t *testing.T) {
	app := nsgotest.Start(t, Root, nsgotest.Size(1280, 800))
	app.Press("label:Open Settings")
	app.WaitFor("id:settings", time.Second)
	app.Press("id:settings >> label:Send usage statistics")
	if v, _ := app.Storage().GetItem(keyStats); v != "off" {
		t.Fatalf("stats = %q, want off", v)
	}
	app.Press("id:settings >> label:Send usage statistics")
	if _, ok := app.Storage().GetItem(keyStats); ok {
		t.Fatal("stats still stored after turning them back on")
	}
}
