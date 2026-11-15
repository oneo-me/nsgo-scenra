package main

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/oneo-me/nsgo-agio/agent"
	"github.com/oneo-me/nsgo-agio/chatview"
	"github.com/oneo-me/nsgo-agio/host"

	"github.com/oneo-me/nsgo-scenra/scenraview"
)

// The storage keys of the agents and models the settings turned off, a
// line each.
const (
	keyDisabled       = "agents.disabled"
	keyDisabledModels = "agents.disabledModels"
)

// skillDir is the scenra skill among the app's resources, which NSGo puts
// in the bundle's Contents/Resources on macOS and beside the executable
// elsewhere, as NSGo names that directory to no app; "" when it is not
// there, as in tests.
func skillDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	root := filepath.Dir(exe)
	if runtime.GOOS == "darwin" {
		root = filepath.Join(root, "..", "Resources")
	}
	dir := filepath.Join(root, "skills", "scenra")
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		return ""
	}
	return dir
}

// lines splits a stored list.
func lines(s string) []string {
	return slices.DeleteFunc(strings.Split(s, "\n"), func(s string) bool { return s == "" })
}

// startAgents runs the coding agents for the window's scope, offering
// each the scenra skill and the server telling what the user sees, and
// keeps the agents and models turned off in the storage. opts.Drivers,
// when set, stands for the agent CLIs, as in tests.
func startAgents(cx *core.Scope, opts host.Options) (*chatview.Model, *scenraview.Server) {
	st := cx.Storage()
	disabled, _ := st.GetItem(keyDisabled)
	models, _ := st.GetItem(keyDisabledModels)
	for _, t := range lines(disabled) {
		opts.Disabled = append(opts.Disabled, agent.Tool(t))
	}
	opts.DisabledModels = lines(models)
	if dir := skillDir(); dir != "" {
		opts.Skills = append(opts.Skills, dir)
	}
	srv, err := scenraview.Listen()
	if err == nil {
		opts.MCPServers = append(opts.MCPServers, srv.Agent())
	}
	h := host.New(opts)
	cx.Cleanup(func() {
		h.Close()
		if srv != nil {
			srv.Close()
		}
	})
	m := chatview.Bind(cx, h)
	cx.Effect(func() {
		var tools, keys []string
		for _, t := range m.State().Tools {
			if t.Disabled {
				tools = append(tools, string(t.Tool))
			}
			for _, id := range t.DisabledModels {
				keys = append(keys, agent.Key(t.Tool, id))
			}
		}
		core.Untracked(cx, func() bool {
			st.SetItem(keyDisabled, strings.Join(tools, "\n"))
			st.SetItem(keyDisabledModels, strings.Join(keys, "\n"))
			return true
		})
	})
	return m, srv
}
