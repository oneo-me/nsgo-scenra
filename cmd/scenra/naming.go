package main

import (
	"context"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/i18n"
	"github.com/nsgo-dev/nsgo/ui"
	agio "github.com/oneo-me/nsgo-agio"
	"github.com/oneo-me/nsgo-agio/chatview"
	"github.com/oneo-me/nsgo-agio/naming"

	"github.com/oneo-me/nsgo-scenra/cmd/scenra/locales"
)

// The storage keys of the OpenAI-compatible model titling conversations.
const (
	keyNamingURL   = "naming.baseURL"
	keyNamingKey   = "naming.apiKey"
	keyNamingModel = "naming.model"
)

// namingTimeout bounds how long a conversation waits for its title.
const namingTimeout = 20 * time.Second

// namer titles each new conversation with the naming model once its first
// message is sent. The host has titled it with that message by then, which
// it keeps without a naming model, when naming fails or once its agent
// renamed it.
type namer struct {
	cx *core.Scope
	m  *chatview.Model
	// fresh are the conversations seen with no message, which a resumed
	// one never is.
	fresh map[string]bool
	// failure is why naming last failed, "" once it succeeds.
	failure *core.Signal[string]
}

func newNamer(cx *core.Scope, m *chatview.Model) *namer {
	n := &namer{cx: cx, m: m, fresh: map[string]bool{}, failure: core.NewSignal(cx, "")}
	cx.Effect(func() {
		s := m.State()
		cx.Untrack(func() { n.look(s) })
	})
	return n
}

// provider is the naming model the settings chose, titling in the
// interface's language.
func (n *namer) provider() naming.Provider {
	st := n.cx.Storage()
	url, _ := st.GetItem(keyNamingURL)
	key, _ := st.GetItem(keyNamingKey)
	model, _ := st.GetItem(keyNamingModel)
	language := "en"
	if strings.HasPrefix(i18n.Language(n.cx), "zh") {
		language = "zh-Hans"
	}
	return naming.Provider{BaseURL: url, APIKey: key, Model: model, TitleLanguage: language}
}

// look starts naming the conversations whose first message was sent since
// it last looked.
func (n *namer) look(s agio.State) {
	fresh := map[string]bool{}
	for _, c := range s.Conversations {
		switch {
		case len(c.Turns) == 0:
			fresh[c.ID] = c.Cursor == "" && !c.Loading
		case n.fresh[c.ID]:
			if p := n.provider(); p.Configured() {
				go n.name(p, c.ID, c.Title, c.Turns[0].Prompt)
			}
		}
	}
	n.fresh = fresh
}

// name asks for a title, which replaces the one taken from the first
// message unless the conversation was renamed since.
func (n *namer) name(p naming.Provider, id, title, prompt string) {
	ctx, cancel := context.WithTimeout(context.Background(), namingTimeout)
	defer cancel()
	named, err := p.Suggest(ctx, prompt)
	if err == nil {
		b := n.m.Backend()
		if c := b.State().Conversation(id); c != nil && c.Title == title {
			err = b.Choose(id, agio.Choice{Title: &named})
		}
	}
	n.cx.Post(func() {
		if p != n.provider() {
			return
		}
		if err != nil {
			n.failure.Set(err.Error())
		} else {
			n.failure.Set("")
		}
	})
}

// namingSection sets the OpenAI-compatible model titling new
// conversations, saying why naming last failed.
func namingSection(cx *core.Scope, n *namer) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	st := cx.Storage()
	field := func(key, title, placeholder string, masked bool) core.Node {
		initial, _ := st.GetItem(key)
		value := core.NewSignal(cx, initial)
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12}, Children: []core.Node{
			core.Text{Content: title, Style: core.Style{FontSize: 13, Width: core.Pt(96), FlexShrink: 0}, StyleFunc: ui.Colored(colors.Text)},
			ui.Input{Value: value.Get, Label: title, Placeholder: placeholder, Masked: masked,
				Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
				OnChange: func(text string) {
					value.Set(text)
					if text = strings.TrimSpace(text); text == "" {
						st.RemoveItem(key)
					} else {
						st.SetItem(key, text)
					}
					n.failure.Set("")
				}},
		}}
	}
	return core.View{Style: core.Style{Gap: 8}, Children: []core.Node{
		core.View{Style: core.Style{Gap: 2}, Children: []core.Node{
			ui.Text(tr.NamingSection(), 15, ui.Semibold, colors.Text),
			core.Text{Content: tr.NamingDetail(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
		}},
		field(keyNamingURL, tr.NamingURL(), naming.DefaultBaseURL, false),
		field(keyNamingKey, tr.NamingKey(), tr.NamingKeyPlaceholder(), true),
		field(keyNamingModel, tr.NamingModel(), tr.NamingModelPlaceholder(), false),
		core.Show{When: func() bool { return n.failure.Get() != "" }, Then: func(cx *core.Scope) core.Node {
			return core.Text{Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Danger),
				ContentFunc: func() string { return tr.NamingFailed(n.failure.Get()) }}
		}},
	}}
}
