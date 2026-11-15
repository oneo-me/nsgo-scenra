# Changelog

## [Unreleased]

- Add Scenra, a flow chart editor whose assistant is a coding agent, such as Claude Code or Codex, run through Agio and taught the format by a bundled skill.
- Add phone, tablet and desktop designs drawn from Tailwind-shaped primitives beside flow charts, one in each `.scenra` file.
- Add an assistant that hears of the selection over MCP, resolves the comments users leave on design elements and works in the open folder to chart its code, saving the user's changes as a message is sent and locking the files while it works, and fixes a file that does not open from its error shown in the workspace.
- Add folder workspaces whose plain-text `.scenra` files show as a tree and one at a time beside a full-height assistant, with a coffee shop example.
- Let designs and components set their own `width=` and `height=`, which the assistant may change, instead of only their device's size.
- Add components, shared pieces of designs kept in the folder's `components/` and drawn with `use`, which the assistant makes and the editor redraws as they change.
- Keep each folder's conversations with the assistant, resumed when the folder opens again and switched or deleted from a menu atop the conversation.
- Play designs as the app would, opening screens, closing them and switching tabs with `tap` and `panel`, marking the tab shown with `selected:` classes, and flash what can be tapped as a design opens or reloads, and from the toolbar.
- Show an element's comment card where the pointer rests instead of on press, and add notes beside comments, which the assistant reads and keeps true.
- Let a file give itself a `name=`, which the assistant may set and the tree, header and window title show instead of its file name.
- Keep a screen's flow as `<name>-flow.scenra` beside its design `<name>.scenra`, and switch between the two from the header.
- Open folders, or Scenra files with their folder, from the Finder on macOS or the command line, save files in place when another file, folder or the window takes their place, and reopen them at the next start.
- Add macOS, Windows and Linux builds in English and Simplified Chinese, for desktops only.
- Add opt-out anonymous usage statistics.
- Add an official website in English and Simplified Chinese on GitHub Pages, with anonymous statistics
- Add `scenraview`, which lets other NSGo apps open, edit and chart folders of Scenra files with the editor's views, alone or in Scenra's window.
