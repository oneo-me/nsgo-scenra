package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestFirstMessageTitlesTheConversation(t *testing.T) {
	var asked atomic.Int32
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model           string
			ReasoningEffort string `json:"reasoning_effort"`
			Messages        []struct{ Content string }
		}
		json.NewDecoder(r.Body).Decode(&req)
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer secret" || req.Model != "small" || req.ReasoningEffort != "low" ||
			len(req.Messages) != 2 || !strings.Contains(req.Messages[1].Content, "coffee") {
			t.Errorf("request %s %q %+v", r.URL.Path, r.Header.Get("Authorization"), req)
		}
		asked.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": `{"title": "Coffee ordering app"}`}}}})
	}))
	defer service.Close()

	app := nsgotest.Start(t, testRoot(t), nsgotest.Size(1280, 800))
	app.Press("id:home-settings")
	app.WaitFor("id:settings", time.Second)
	app.Type("id:settings >> label:Base URL", service.URL+"/v1")
	app.Type("id:settings >> label:API key", "secret")
	app.Type("id:settings >> label:Model", " small ")
	if v, _ := app.Storage().GetItem(keyNamingModel); v != "small" {
		t.Fatalf("model = %q, want small", v)
	}
	app.Press("id:naming-effort")
	app.Press("id:naming-effort-1")
	if v, _ := app.Storage().GetItem(keyNamingEffort); v != "low" {
		t.Fatalf("effort = %q, want low", v)
	}

	dir := filepath.Join(t.TempDir(), "coffee")
	os.Mkdir(dir, 0o755)
	app.PickDirectories = func(core.DirectoriesRequest) []core.Directory { return []core.Directory{{Path: dir}} }
	app.Press("id:home-recent")
	app.Press("id:home-open-folder")
	app.WaitFor("id:files", 5*time.Second)
	app.WaitUntil(5*time.Second, func() bool { return len(app.Query("id:composer")) > 0 })
	app.Type("id:composer", "Design the screens of an app for ordering coffee")
	app.Press("role:button Send")
	app.WaitUntil(5*time.Second, func() bool { return app.Find("id:chat-title").Text == "Coffee ordering app" })
	if n := asked.Load(); n != 1 {
		t.Fatalf("asked %d times, want 1", n)
	}
}

func TestNoNamingModelKeepsTheFirstLine(t *testing.T) {
	app := nsgotest.Start(t, testRoot(t), nsgotest.Size(1280, 800))
	dir := filepath.Join(t.TempDir(), "coffee")
	os.Mkdir(dir, 0o755)
	app.PickDirectories = func(core.DirectoriesRequest) []core.Directory { return []core.Directory{{Path: dir}} }
	app.Press("id:home-recent")
	app.Press("id:home-open-folder")
	app.WaitFor("id:files", 5*time.Second)
	app.WaitUntil(5*time.Second, func() bool { return len(app.Query("id:composer")) > 0 })
	app.Type("id:composer", "Design the screens\nof an app")
	app.Press("role:button Send")
	app.WaitUntil(5*time.Second, func() bool { return app.Find("id:chat-title").Text == "Design the screens" })
}
