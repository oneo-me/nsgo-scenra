# Scenra architecture

Scenra is a single-window [NSGo](https://github.com/nsgo-dev/nsgo) desktop app, for desktops only, built from `scenraview`, which gives other NSGo apps the same support for Scenra files, as Agio's `chatview` does for conversations. The editor works in one folder, whose Scenra files the sidebar shows as a tree; each file is one design of a screen or one flow chart. One file at a time shows as the document, with its undo history, under a header naming it, beside the assistant running the window's height, whose one conversation is about the folder. The design and diagram draw it.

## Code map

| Path | Responsibility |
| --- | --- |
| `flowfile/` | The Scenra file format: a design's elements or a flow's nodes and edges, parsing, canonical formatting and edits. Its package comment is the format's reference. |
| `scenraview/` | The `Editor` an app opens a folder of Scenra files in, and its views: `Window`, or the `Header`, `Sidebar`, `Workspace` and `Assistant` it lays out. |
| `scenraview/editor.go`, `folder.go`, `header.go`, `diagram.go` | The open folder and the files found in it, the document shown and its history, the header switching a design and its flow, and the flow chart. |
| `scenraview/sidebar.go`, `mockup.go`, `icons.go`, `comments.go` | The tree of the folder's files and the new file dialog, the drawing of a design and the comments on its elements. |
| `scenraview/assistant.go`, `files.go` | The conversation about the folder, run by an Agio backend, and the files read again as agents write them. |
| `scenraview/server.go`, `selection.go` | The MCP server telling agents what the user sees of a file. |
| `scenraview/locales/` | The views' messages, which an app's catalogs may supply other languages of. |
| `cmd/scenra/` | Scenra, the app: the window, menu, the agents it runs and their settings, opt-out usage statistics, `nsgo.yaml`, assets and its own `locales/`. |
| `cmd/scenra/resources/skills/scenra/` | The skill teaching agents the format, which ships among the app's resources. |
| `website/` | The official static site, in English and Simplified Chinese, deployed to GitHub Pages. |

## Boundaries

- `flowfile` never imports NSGo, so the format stays testable on its own.
- The assistant is a coding agent run through [Agio](https://github.com/oneo-me/nsgo-agio), not a model client of Scenra's own, so it brings the user's accounts and can read the project. Agents edit the file with their own tools, as the skill teaches, rather than through drawing tools: plain text is what they edit best. What they cannot see, the selection, the open screen and flow, the comments and the file's problems, the editor's `Server` tells them over MCP.
- `scenraview` is product-neutral: the host app gives it the Agio backend, the server and skill its agents get, where to set them up, its toaster and the storage prefix that reopens files, and keeps menus, settings, statistics and the window to itself. It ships no app catalog, so it never sets the app's languages.
- A file holds one design or one flow, named by the file, so a project is a folder the agent grows file by file and the tree is its outline; there are no titles, screens or groups inside a file.
- A screen's flow is `<name>-flow.scenra` beside its design `<name>.scenra`, a convention of names rather than a link in either file, so the tree keeps them together and the header switches between them without the format knowing; a design's name therefore never ends in `-flow`.
- Only one document is open, so leaving it, for another file or folder or by closing the window, saves it instead of asking; undo history goes with it.
- The folder is scanned off the UI's goroutine, leaving out hidden folders and `node_modules` and stopping after a limit, so a large folder stays responsive; a file the agent makes in its turn opens.
- Every change to a document goes through the editor, which records it for undo. What an agent writes in one turn undoes together.
- The files are the only channel between the user and the agent, so sending a message saves the user's changes, and the user cannot change the documents while the agent works. A file an agent leaves unparsable keeps the last version that parsed, and the server tells the agent why.
- The skill is a resource, not embedded, so agents read it from the app's package; NSGo names no resource directory, so the app finds it where NSGo's packages put resources.
- A design is a draft drawn from a few primitives shaped by Tailwind CSS classes, not from ready-made controls, so the agent can draw any look. Classes NSGo does not know reach the agent through the server.
- A design lays out at its device's size on a plane the design canvas moves and scales, as the flow chart's camera does, since NSGo's zooming scroll views exist only on Apple platforms.
- Platforms hit-test views without that transform, so a layer over the design finds the element under the pointer from the elements' frames, as the flow chart finds its nodes. Elements are named by their paths, which the selection and the server use.
- A comment is an attribute of its element in the flow file, so it is saved with the file and follows the element through edits, where paths shift. The user adds and removes comments; the skill tells agents to remove one only once it is resolved.
- The flow file names icons by their Lucide names, or older ones it maps to them, as `flowfile` cannot import NSGo. `tools/iconsgen`, run by `go generate` after updating NSGo, writes every name NSGo's Lucide package has into `flowfile` and the map from names to icons into `scenraview`.
- One flow view follows the document shown instead of remounting per file, and the design covers it while a design shows.
- The chart fits only once a change's batch is over: remounting a chart, or fitting it while it takes new data, trips NSGo's commit check on macOS.
- The folder has one conversation, whose agent works in it with Agio's full access there, as a design and its flows are made together; opening another folder starts another.
- User-visible words live in `locales/`, in both languages.
- Statistics never carry document content, messages or paths. `nsgo dev` builds and test binaries send none.

## Core flows

**Assistant turn.** Sending a message saves the documents and sends the message to the folder's conversation, whose agent asks the server what the user sees, reads and writes the files and makes new ones. The editor looks at the open files a few times a second and reads a changed one again into its document as a change of the running turn; the server's answer reads them and scans the folder first, so the agent hears of what it just wrote.

**Files.** The app opens a folder, from the picker, the Finder or the command line; opening a file opens the folder holding it unless it is in the open one. Choosing a file in the tree, or its pair in the header, saves the document shown and parses the file in its place. A new file is named in a dialog and made in the folder selected in the tree; a new flow takes the shown design's name. A document saves to the path it was opened from, in the canonical form, so `#` comments in a hand-written file are dropped.

**Statistics.** Release builds send an Umami pageview at start, an event per message sent and heartbeats while the chart or the conversation changes. A queue on its own goroutine posts them and drops failures.

## Data and state

Settings, the open folder, the path of the file shown and the install's random statistics ID live in NSGo's storage on the device. Documents live only in the folder's files.

## Verification

`go tool nsgo test` drives `scenraview`'s views and the app headless against a scripted Agio driver whose agent writes the file and calls the server, and covers the format and the skill's coverage of it. Real agents need manual review. Check layout and platform behavior by hand with `go tool nsgo dev cmd/scenra`, or with `go tool nsgo test -target macos`.

## Subjects

| Document | When to read |
| --- | --- |
| [`flowfile` package comment](flowfile/) | Changing the Scenra file format |
