# Changelog

## [Unreleased]

- Add Scenra, a flow chart editor whose assistant is a coding agent, such as Claude Code or Codex, run through Agio and taught the format by a bundled skill.
- Add phone, tablet and desktop designs drawn from Tailwind-shaped primitives beside flow charts, one in each `.scenra` file.
- Add an assistant that hears of the selection over MCP, resolves the comments users leave on design elements and works in the open folder to chart its code, saving the user's changes as a message is sent and locking the files while it works, and fixes a file that does not open from its error shown in the workspace.
- Add folder workspaces whose plain-text `.scenra` files sit at the folder's top, components in `components/`, listed in one-level groups their names give, such as `Orders/Detail`, made from the File menu, and one at a time, named after the folder in the title bar, beside a full-height assistant, both hidden with ⌘B to give the canvas the window, with a coffee shop example.
- Let designs and components set their own `width=` and `height=`, which the assistant may change, instead of only their device's size.
- Add components, shared pieces of designs kept in the folder's `components/` and drawn with `use`, whose classes go to the component's top element as React's `className` does, which the assistant makes, leaving their taps to the designs using them, and the editor redraws as they change.
- Show every component of the folder together, in their name groups, from **Components** atop the list or the View menu, each as large as its content unless it sets a width or `width=fill`, outlined on its `backdrop=` rather than on a card, beside the `variant` states its file declares, scaling small ones up to a least area.
- Let uses pick their component's variants by ID with `variant=`, which may start from each other, put it in states with `state=` that apply its own `selected:`, `hover:` and `group-*:` classes, and set the panels, notes, variants and states of its named elements, and the classes of its slots by their own names.
- Let a design fill another's `outlet=` with `in=`, drawn at the size the outlet lays out at and played inside its host, so a window's sidebar stays, marks what shows and swaps only its content.
- Let components show at a `preview=` width and on a `backdrop=` of layers over the theme's background.
- Tell agents where each element lays out with a `get_layout` tool, of a whole design or one element's levels, have `get_view` list all notes, the open file's or none, and report text cut off, elements outside their design or of no size, `w-full` filling nothing inside its holder's padding and borders, uses far narrower than their components, variants and outlets that are missing or loop, and uses that could pick a variant.
- Show beside each file in a Git repository the lines it added and removed since the last commit.
- Show the folder's own pictures, kept in its `assets/` and named by an image's `src=`, which the assistant copies there and the editor redraws as they change.
- Let a use change the texts, icons, classes, pictures, taps and nested components of the elements its component names, as Figma's instances and Sketch's symbols do, and fill its component's slots with elements of its own, which components may pass on, reporting loops of components through their fills by the chain of uses.
- Keep each folder's conversations with the assistant, resumed when the folder opens again and switched or deleted from a menu atop the conversation, and title new ones from their first message with an OpenAI-compatible model and reasoning effort set in the settings.
- Play designs as the app would, opening screens, closing them, or the window a page shows in, and switching tabs with `tap` and `panel`, marking the tab shown with `selected:` classes, and flash what can be tapped as a design opens or reloads, and from the toolbar.
- Show an element's comment card where the pointer rests instead of on press, and add notes beside comments, which the assistant reads and keeps true.
- Let a file give itself a `name=`, which the assistant may set and the sidebar, header and window title show instead of its file name.
- Keep a screen's flow as `<name>-flow.scenra` beside its design `<name>.scenra`, and switch between the two from the header.
- Let a file hold several designs or several flows, each with an ID and opened by taps such as `open #sign-up`, packed with the least margin `gap=` points apart, which the assistant sets to suit their sizes.
- Open folders, or Scenra files with their folder, from the Finder on macOS or the command line, save files in place when another file, folder or the window takes their place, and reopen them at the next start.
- Add macOS, Windows and Linux builds in English and Simplified Chinese, for desktops only.
- Add opt-out anonymous usage statistics.
- Add an official website in English and Simplified Chinese on GitHub Pages, with anonymous statistics
- Add `scenraview`, which lets other NSGo apps open, edit and chart folders of Scenra files with the editor's views, alone or in Scenra's window.
