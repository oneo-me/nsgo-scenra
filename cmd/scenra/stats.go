package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"

	"github.com/oneo-me/nsgo-scenra/scenraview"
)

// The app counts its use in Umami under a site of its own, as NSGo's
// gallery does: a pageview at start, an event for each message sent and
// heartbeats while it is used, tagged with the platform and carrying a
// random ID kept in the storage, so that one install is one visitor. No
// chart, message or path is sent. Builds of nsgo dev (statsBuild) and test
// binaries report nothing, and the settings turn it off.
const (
	statsURL     = "https://umami.oneo.me/api/send"
	statsWebsite = "0d6c7a62-255c-498f-ad36-c4c57d93ab0c"
	statsHost    = "scenra.oneo.me"
)

// Storage keys of the statistics: the install's ID, and keyStats "off"
// when the settings turned them off.
const (
	keyInstall = "installID"
	keyStats   = "stats"
)

// Umami measures a visit from its first event to its last, so while the
// app is used it sends a heartbeat each statsBeat, for statsIdle after the
// chart last changed or a message was last sent: a window left open does
// not count as use.
const (
	statsBeat = time.Minute
	statsIdle = 5 * time.Minute
)

// statsOn reports whether the settings leave the statistics on.
func statsOn(cx *core.Scope) bool {
	v, _ := cx.Storage().GetItem(keyStats)
	return v != "off"
}

// setStatsOn turns the statistics on or off.
func setStatsOn(cx *core.Scope, on bool) {
	if on {
		cx.Storage().RemoveItem(keyStats)
	} else {
		cx.Storage().SetItem(keyStats, "off")
	}
}

// statsPayload is an event of Umami's /api/send: a pageview without a
// name.
type statsPayload struct {
	Website  string `json:"website"`
	Hostname string `json:"hostname"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Language string `json:"language,omitempty"`
	Screen   string `json:"screen,omitempty"`
	Tag      string `json:"tag"`
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
}

// usage reports the app's use: the pageview, the events and the
// heartbeats. Events are made on the graph's goroutine and sent from
// another, oldest first; one that fails is dropped.
type usage struct {
	cx *core.Scope
	// send sends an event of name, the pageview for "", and nil drops it.
	send func(name string)
	// since is when the app was last used.
	since time.Time
}

// newUsage is the app's usage, which reports nothing in nsgo dev's
// builds or tests.
func newUsage(cx *core.Scope) *usage {
	u := &usage{cx: cx, since: time.Now()}
	if !statsBuild || testing.Testing() {
		return u
	}
	store := cx.Storage()
	id, ok := store.GetItem(keyInstall)
	if !ok {
		b := make([]byte, 16)
		rand.Read(b)
		id = hex.EncodeToString(b)
		store.SetItem(keyInstall, id)
	}
	env := cx.Env()
	agent := statsAgent(env.OS())
	queue := make(chan statsPayload, 32)
	go sendStats(queue, agent)
	u.send = func(name string) {
		if !statsOn(cx) {
			return
		}
		w := env.Window()
		p := statsPayload{Website: statsWebsite, Hostname: statsHost, URL: "/", Title: "Scenra",
			Language: env.Locale(), Screen: fmt.Sprintf("%dx%d", int(w.Width), int(w.Height)),
			Tag: osName(env.OS()), ID: id, Name: name}
		select {
		case queue <- p:
		default:
		}
	}
	return u
}

// start sends the pageview, and the heartbeats while the documents the
// editors watch shows change or messages are sent.
func (u *usage) start() {
	if u.send == nil {
		return
	}
	cx := u.cx
	u.send("")
	var beat func()
	beat = func() {
		if time.Since(u.since) < statsIdle {
			u.send("heartbeat")
		}
		cx.AfterFunc(statsBeat, beat)
	}
	cx.AfterFunc(statsBeat, beat)
}

// watch counts the changes of e's documents as use, while cx lasts.
func (u *usage) watch(cx *core.Scope, e *scenraview.Editor) {
	if u.send == nil {
		return
	}
	cx.Effect(func() {
		e.File()
		core.Untracked(cx, func() bool { u.since = time.Now(); return true })
	})
}

// sent counts a message sent to the assistant.
func (u *usage) sent() {
	if u.send == nil {
		return
	}
	u.since = time.Now()
	u.send("message")
}

// sendStats posts the queue's events to Umami until it closes.
func sendStats(queue <-chan statsPayload, agent string) {
	client := &http.Client{Timeout: 10 * time.Second}
	for p := range queue {
		body, _ := json.Marshal(map[string]any{"type": "event", "payload": p})
		req, err := http.NewRequest(http.MethodPost, statsURL, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if agent != "" {
			req.Header.Set("User-Agent", agent)
		}
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
		}
	}
}

// statsAgent is the User-Agent the app sends, as a browser on the
// platform would say it, for Umami to tell the system and not take it for
// a bot as it does Go's own.
func statsAgent(os core.OS) string {
	platforms := map[core.OS]string{
		core.OSMacOS:   "Macintosh; Intel Mac OS X 10_15_7",
		core.OSWindows: "Windows NT 10.0; Win64; x64",
		core.OSLinux:   "X11; Linux x86_64",
	}
	p, ok := platforms[os]
	if !ok {
		return ""
	}
	return "Mozilla/5.0 (" + p + ") Scenra/1.0"
}

// osName is the platform's name, as the statistics tag it.
func osName(os core.OS) string {
	names := map[core.OS]string{core.OSMacOS: "macOS", core.OSWindows: "Windows", core.OSLinux: "Linux"}
	if n, ok := names[os]; ok {
		return n
	}
	return "unknown"
}
