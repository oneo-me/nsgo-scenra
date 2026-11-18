//go:build !scenrademo

package main

import (
	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/oneo-me/nsgo-agio/chatview"
)

// demoBuild is whether this is a demo build, demo.go's, which stands for
// the agents with a scripted conversation.
const demoBuild = false

func demoOptions() []nsgo.Option             { return nil }
func demoOpen(*tabs)                         {}
func demoAgents(*core.Scope) *chatview.Model { return nil }
