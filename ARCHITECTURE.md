# Scenra architecture

Scenra is a single-window [NSGo](https://github.com/nsgo-dev/nsgo) desktop app, for desktops only, built from `scenraview`, which gives other NSGo apps the same support for Scenra files, as Agio's `chatview` does for conversations. The editor holds the open documents, a tab each across the window's top, beside the assistant running the window's height; a document is an app's screens each with a design and its flow charts, with its file, undo history and conversation. The sidebar, design and diagram draw the document shown, and the assistant runs a model conversation whose tool calls edit the document it began on.

## Code map

| Path | Responsibility |
| --- | --- |
| `flowfile/` | The flow file format: screens, interface elements and flows, parsing, canonical formatting and edits. Its package comment is the format's reference. |
| `scenraview/` | The `Editor` an app opens Scenra files in, and its views: `Window`, or the `TabBar`, `Sidebar`, `Workspace` and `Assistant` it lays out. |
| `scenraview/editor.go`, `diagram.go` | The open documents, their history, files and open screen and flow, and the flow chart. |
| `scenraview/screens.go`, `mockup.go`, `icons.go`, `comments.go` | The sidebar of designs and flows, the drawing of a screen's design and the comments on its elements. |
| `scenraview/assistant.go`, `files.go` | The conversation about each document, run by an Agio backend, and the files read again as agents write them. |
| `scenraview/server.go`, `selection.go` | The MCP server telling agents what the user sees of a file. |
| `scenraview/locales/` | The views' messages, which an app's catalogs may supply other languages of. |
| `cmd/scenra/` | Scenra, the app: the window, menu, the agents it runs and their settings, opt-out usage statistics, `nsgo.yaml`, assets and its own `locales/`. |
| `cmd/scenra/resources/skills/scenra/` | The skill teaching agents the format, which ships among the app's resources. |
| `website/` | The official static site, in English and Simplified Chinese, deployed to GitHub Pages. |

## Boundaries

- `flowfile` never imports NSGo, so the format stays testable on its own.
- The assistant is a coding agent run through [Agio](https://github.com/oneo-me/nsgo-agio), not a model client of Scenra's own, so it brings the user's accounts and can read the project. Agents edit the file with their own tools, as the skill teaches, rather than through drawing tools: plain text is what they edit best. What they cannot see, the selection, the open screen and flow, the comments and the file's problems, the editor's `Server` tells them over MCP.
- `scenraview` is product-neutral: the host app gives it the Agio backend, the server and skill its agents get, where to set them up, its toaster and the storage prefix that reopens files, and keeps menus, settings, statistics and the window to itself. It ships no app catalog, so it never sets the app's languages.
- Every change to the document goes through the editor, which records it for undo. What an agent writes in one turn undoes together.
- The file is the only channel between the user and the agent, so sending a message saves the user's changes, and the user cannot change the document while its agent works. A file an agent leaves unparsable keeps the last version that parsed, and the server tells the agent why.
- The skill is a resource, not embedded, so agents read it from the app's package; NSGo names no resource directory, so the app finds it where NSGo's packages put resources.
- A design is a draft drawn from a few primitives shaped by Tailwind CSS classes, not from ready-made controls, so the agent can draw any look. Classes NSGo does not know reach the agent through the server.
- A design lays out at its device's size on a plane the design canvas moves and scales, as the flow chart's camera does, since NSGo's zooming scroll views exist only on Apple platforms.
- Platforms hit-test views without that transform, so a layer over the design finds the element under the pointer from the elements' frames, as the flow chart finds its nodes. Elements are named by their paths, which the selection and the server use.
- A comment is an attribute of its element in the flow file, so it is saved with the file and follows the element through edits, where paths shift. The user adds and removes comments; the skill tells agents to remove one only once it is resolved.
- The flow file names icons by their Lucide names, or older ones it maps to them, as `flowfile` cannot import NSGo. `tools/iconsgen`, run by `go generate` after updating NSGo, writes every name NSGo's Lucide package has into `flowfile` and the map from names to icons into `scenraview`.
- One flow view follows the open flow of the document shown instead of remounting per flow or tab, and the design covers it while its tab is open.
- The chart fits only once a change's batch is over: remounting a chart, or fitting it while it takes new data, trips NSGo's commit check on macOS.
- Each document has its own conversation, whose agent works in the folder holding the file with Agio's full access there; a turn keeps editing its document while another tab shows.
- User-visible words live in `locales/`, in both languages.
- Statistics never carry document content, messages or paths. `nsgo dev` builds and test binaries send none.

## Core flows

**Assistant turn.** Sending a message saves the document and sends the message to its conversation, whose agent reads the file, asks the server what the user sees, and writes the file. The editor looks at the open files a few times a second and reads a changed one again into the document as a change of the running turn; the server's answer reads it first, so the agent hears of what it just wrote.

**Files.** Every document has a file: making one asks where to save it first, and the assistant offers to make or open one while none is. Opening parses a flow file into a tab of its own, or shows the file's tab when it is open already; a file without screens, from before screens existed, reads as one screen with one flow. A document saves to the path it was opened from or saved to, which NSGo's save dialog reports. Saving writes the canonical form, so `#` comments in a hand-written file are dropped.

**Statistics.** Release builds send an Umami pageview at start, an event per message sent and heartbeats while the chart or the conversation changes. A queue on its own goroutine posts them and drops failures.

## Data and state

Settings, the paths of the files open in tabs and the install's random statistics ID live in NSGo's storage on the device. Documents live only in the files the user opens and saves.

## Verification

`go tool nsgo test` drives `scenraview`'s views and the app headless against a scripted Agio driver whose agent writes the file and calls the server, and covers the format and the skill's coverage of it. Real agents need manual review. Check layout and platform behavior by hand with `go tool nsgo dev cmd/scenra`, or with `go tool nsgo test -target macos`.

## Subjects

| Document | When to read |
| --- | --- |
| [`flowfile` package comment](flowfile/) | Changing the flow file format |
