//go:build scenrademo

// A demo build, go build -tags scenrademo ./cmd/scenra, is Scenra showing
// a scripted conversation instead of running agents, for the website's
// screenshots and for reviewing the app without an agent CLI. It reports
// no use. The website's screenshot script runs it with a HOME of its own
// and the files to show.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	agio "github.com/oneo-me/nsgo-agio"
	"github.com/oneo-me/nsgo-agio/agent"
	"github.com/oneo-me/nsgo-agio/chatview"
	"github.com/oneo-me/nsgo-agio/quota"
	"github.com/oneo-me/nsgo-agio/transcript"
)

const demoBuild = true

// SCENRA_DEMO_OPEN lists the files the demo opens, the last shown,
// SCENRA_DEMO_APPEARANCE light or dark forces the appearance, and
// SCENRA_DEMO_LANG=zh writes the conversation in Chinese. They are read
// once at start.
var demoWords = demoEnglish

func init() {
	if os.Getenv("SCENRA_DEMO_LANG") == "zh" {
		demoWords = demoChinese
	}
}

// demoOptions are the window's options the demo adds.
func demoOptions() []nsgo.Option {
	switch os.Getenv("SCENRA_DEMO_APPEARANCE") {
	case "light":
		return []nsgo.Option{nsgo.Appearance(core.AppearanceLight)}
	case "dark":
		return []nsgo.Option{nsgo.Appearance(core.AppearanceDark)}
	}
	return nil
}

// demoOpen opens the files SCENRA_DEMO_OPEN lists, a second apart, as
// a flow chart opened first in its folder draws nothing.
func demoOpen(t *tabs) {
	paths := filepath.SplitList(os.Getenv("SCENRA_DEMO_OPEN"))
	if len(paths) == 0 {
		return
	}
	open := func(path string) {
		if err := t.openPath(path); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	open(paths[0])
	go func() {
		for _, path := range paths[1:] {
			time.Sleep(time.Second)
			t.cx.Post(func() { open(path) })
		}
	}()
}

// demoText is what the conversation says, in one language.
type demoText struct {
	Title                    string
	Ask, Thinking, Answer    string
	FollowUp, FollowThinking string
	FollowAnswer             string
}

var demoEnglish = demoText{
	Title:    "Now playing on desktop",
	Ask:      "Design the desktop app's now playing screen: the cover large on the left, synced lyrics on the right, dark and calm.",
	Thinking: "The phone's now playing screen already sets the look: a dark backdrop, the cover, the controls under it. On a desktop the lyrics get a column of their own.",
	Answer: "Drew **Now playing** for the desktop app:\n\n" +
		"- The cover, the title and the controls on the left, under a soft blue glow.\n" +
		"- The lyrics on the right, the current line bright and centered, the others fading out.\n" +
		"- Up next, sharing and volume along the bottom.\n\n" +
		"Tapping the player bar's cover opens it, and the chevron closes it again. I left notes on the lyrics and the next button.",
	FollowUp:       "Chart how someone gets from Discover to it.",
	FollowThinking: "A flow of its own, from opening the app to the player, with the choice made there.",
	FollowAnswer:   "Added **Desktop: from home to the player** to the flows: the player bar opens the player from any page, then you skip or collapse it.",
}

var demoChinese = demoText{
	Title:    "桌面端正在播放",
	Ask:      "设计桌面端的正在播放页：左边是大封面，右边是同步歌词，深色、安静一些。",
	Thinking: "手机端的正在播放页已经定下了风格：深色背景、封面、下面是控制按钮。桌面端可以给歌词单独一栏。",
	Answer: "画好了桌面端的**正在播放**：\n\n" +
		"- 左边是封面、歌名和控制按钮，衬着柔和的蓝色光晕。\n" +
		"- 右边是歌词，当前行居中高亮，上下行渐隐。\n" +
		"- 底部是下一首、分享和音量。\n\n" +
		"点按播放条的封面打开它，左上角的箭头收起。歌词和下一首按钮上留了备注。",
	FollowUp:       "画一下从发现页到这里的流程。",
	FollowThinking: "单独一张流程图，从打开应用到播放页，以及在那里的选择。",
	FollowAnswer:   "在流程里加了**桌面端：从首页到播放**：任何页面都能从播放条打开播放页，然后切歌或收起。",
}

// demoAgents stands for the agents with a scripted backend, whose first
// conversation has already drawn the folder's now playing screen.
func demoAgents(cx *core.Scope) *chatview.Model {
	return chatview.Bind(cx, &demoBackend{state: agio.State{Tools: []agio.Tool{{
		Tool: agent.Claude, Path: "/Users/you/.local/bin/claude", Version: "2.1.7",
		Catalog: agent.Catalog{Model: "opus", Effort: "high", Models: []agent.Model{{
			Tool: agent.Claude, ID: "opus", Name: "Claude Opus", Efforts: []string{"low", "medium", "high"},
			DefaultEffort: "high", ContextWindow: 200000, Images: true, Default: true,
		}}},
	}, {
		Tool: agent.Codex, Path: "/opt/homebrew/bin/codex", Version: "0.58.0",
		Catalog: agent.Catalog{Model: "gpt-5.4-codex", Models: []agent.Model{{
			Tool: agent.Codex, ID: "gpt-5.4-codex", Name: "GPT-5.4 Codex", ContextWindow: 400000,
		}}},
	}}}})
}

// demoTurn is the exchange that drew the screen.
func demoTurn(w demoText, start time.Time) *transcript.Turn {
	t := transcript.New("turn-1", w.Ask, nil, start)
	t.Apply(agent.Event{Type: agent.ThinkingDelta, Segment: "think", Text: w.Thinking})
	t.Apply(agent.Event{Type: agent.ThinkingDone, Segment: "think", Text: w.Thinking})
	t.Apply(agent.Event{Type: agent.ToolStart, Segment: "view", Tool: "scenra_view", Preview: "mobile-now-playing.scenra"})
	t.Apply(agent.Event{Type: agent.ToolEnd, Segment: "view", Tool: "scenra_view"})
	t.Apply(agent.Event{Type: agent.ToolStart, Segment: "read", Tool: "read", Preview: "components/player-bar.scenra"})
	t.Apply(agent.Event{Type: agent.ToolEnd, Segment: "read", Tool: "read"})
	t.Apply(agent.Event{Type: agent.ToolStart, Segment: "write", Tool: "write", Preview: "desktop-now-playing.scenra"})
	t.Apply(agent.Event{Type: agent.ToolEnd, Segment: "write", Tool: "write"})
	t.Apply(agent.Event{Type: agent.TextDone, Segment: "final", Text: w.Answer})
	t.Finish(transcript.Completed, start.Add(48*time.Second))
	return t
}

// demoFollowUp is the second exchange, which charted the flow.
func demoFollowUp(w demoText, start time.Time) *transcript.Turn {
	t := transcript.New("turn-2", w.FollowUp, nil, start)
	t.Apply(agent.Event{Type: agent.ThinkingDelta, Segment: "think-2", Text: w.FollowThinking})
	t.Apply(agent.Event{Type: agent.ThinkingDone, Segment: "think-2", Text: w.FollowThinking})
	t.Apply(agent.Event{Type: agent.ToolStart, Segment: "write-2", Tool: "write", Preview: "flows/desktop-home-to-player.scenra"})
	t.Apply(agent.Event{Type: agent.ToolEnd, Segment: "write-2", Tool: "write"})
	t.Apply(agent.Event{Type: agent.TextDone, Segment: "final-2", Text: w.FollowAnswer})
	t.Finish(transcript.Completed, start.Add(21*time.Second))
	return t
}

// demoBackend is a scripted agio.Backend: nothing runs, and the first
// conversation made, the folder's, comes with the scripted turns.
type demoBackend struct {
	mu       sync.Mutex
	state    agio.State
	watchers []func()
}

func (b *demoBackend) State() agio.State {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.state
	s.Conversations = slices.Clone(s.Conversations)
	return s
}

func (b *demoBackend) Watch(fn func()) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.watchers = append(b.watchers, fn)
	return func() {}
}

func (b *demoBackend) Create(c agio.Create) (string, error) {
	now := time.Now()
	b.mu.Lock()
	id := fmt.Sprintf("demo-%d", len(b.state.Conversations))
	conv := &agio.Conversation{ID: id, Title: c.Title, Dir: c.Dir, Model: agent.Key(agent.Claude, "opus"), Effort: "high",
		Created: now, Updated: now}
	if len(b.state.Conversations) == 0 {
		w := demoWords
		conv.Title, conv.Bound = w.Title, true
		conv.Turns = []*transcript.Turn{demoTurn(w, now.Add(-6*time.Minute)), demoFollowUp(w, now.Add(-2*time.Minute))}
		conv.ContextTokens, conv.ContextWindow = 24800, 200000
	}
	b.state.Conversations = append([]*agio.Conversation{conv}, b.state.Conversations...)
	watchers := slices.Clone(b.watchers)
	b.mu.Unlock()
	for _, fn := range watchers {
		fn()
	}
	return id, nil
}

func (b *demoBackend) Send(string, agio.Message) error   { return nil }
func (b *demoBackend) Stop(string) error                 { return nil }
func (b *demoBackend) Choose(string, agio.Choice) error  { return nil }
func (b *demoBackend) Remove(string) error               { return nil }
func (b *demoBackend) Discover()                         {}
func (b *demoBackend) Enable(agent.Tool, bool) error     { return nil }
func (b *demoBackend) EnableModels([]string, bool) error { return nil }
func (b *demoBackend) RefreshQuota(quota.Provider)       {}
func (b *demoBackend) Close() error                      { return nil }
