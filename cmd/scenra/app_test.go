package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"github.com/oneo-me/nsgo-agio/agent"
	"github.com/oneo-me/nsgo-agio/host"
)

// TestMain gives the platform the main thread under nsgo test -target.
func TestMain(m *testing.M) { nsgotest.Main(m) }

// fake is an agent CLI that offers a model and starts no session.
type fake struct{}

func (fake) Tool() agent.Tool { return agent.Claude }
func (fake) Catalog(context.Context, string) (agent.Catalog, error) {
	return agent.Catalog{Model: "opus", Models: []agent.Model{{Tool: agent.Claude, ID: "opus", Name: "Opus"}}}, nil
}
func (fake) Start(agent.StartOptions, func(agent.Event), func(error)) (agent.Session, agent.Info, error) {
	return nil, agent.Info{}, errors.New("no sessions")
}
func (fake) History(string, string) ([]agent.Event, error) { return nil, nil }

// testRoot is the app with the fake agent.
func testRoot(t *testing.T) func(cx *core.Scope) core.Node {
	home := t.TempDir()
	return func(cx *core.Scope) core.Node {
		return root(cx, host.Options{Env: []string{}, Home: home, Tools: []agent.Tool{agent.Claude}, Drivers: map[agent.Tool]agent.Driver{agent.Claude: fake{}}})
	}
}

func TestSettingsTurnAgentsOff(t *testing.T) {
	app := nsgotest.Start(t, testRoot(t), nsgotest.Size(1280, 800))
	path := filepath.Join(t.TempDir(), "hello.scenra")
	app.SaveFile = func(core.SaveRequest) string { return path }
	app.Press("id:chat-new-file")
	app.WaitFor("id:title-bar >> label:hello.scenra", 5*time.Second)
	app.Press("id:open-settings")
	app.WaitFor("id:settings", time.Second)
	app.WaitFor("id:settings >> text:Claude Code", 5*time.Second)
	app.Press("id:settings >> role:switch >> nth=0")
	app.WaitUntil(5*time.Second, func() bool { v, _ := app.Storage().GetItem(keyDisabled); return v == "claude" })
}

func TestSettingsTurnOffTheStatistics(t *testing.T) {
	app := nsgotest.Start(t, testRoot(t), nsgotest.Size(1280, 800))
	app.Press("id:open-settings")
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
