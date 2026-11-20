# Changelog

## [Unreleased]

- Add `scenraview.Check`, telling the problems of a folder's or a file's Scenra files, as the MCP server reports them, without opening Scenra.
- Draw any design or view inside another with `use` by its name, leaving `components/` to the shared components.
- Shape a design's frame, its background, outline and corners, with `class=` on its line, so the assistant no longer draws a second frame inside it.
- Fix a view filling an outlet flashing at its device's size, a large rounded phone frame, before taking its outlet's.

## [0.1.1] - 2026-11-18

- Add a language setting choosing English, Simplified Chinese or the system's language, which applies at once and is kept for later starts.
- Redesign the website after Agio's, with screenshots of a music app's designs and flows in both languages and appearances.
- Fix a comment card opening at the corner of the icon or text pressed instead of beside the pointer.

## [0.1.0] - 2026-11-18

Scenra's first release: a desktop designer whose assistant, a coding agent you already use, draws an app's screens and flow charts in plain-text files.

- Add folder workspaces of plain-text `.scenra` files holding phone, tablet and desktop designs drawn from Tailwind-shaped primitives, or flow charts.
- Add an assistant run through Agio by Claude Code, Codex or another coding agent, which works on the selection, resolves comments and keeps each folder's conversations.
- Add components shared by designs, with variants, states, overrides and slots, shown together from **Components**.
- Add outlets filling a window's content with another design, and play designs as the app would through their taps.
- Add comments and notes on design elements, pictures from the folder's `assets/`, and each file's Git changes in the sidebar.
- Add a tab per folder beside a Home tab of recent folders and the settings, reopened at the next start.
- Add macOS, Windows and Linux builds in English and Simplified Chinese, which update themselves.
