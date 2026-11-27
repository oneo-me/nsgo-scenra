# Changelog

## [Unreleased]

- Open the settings in a dialog from a gear at the end of the title bar, over any tab, leaving the Home tab to the recent folders.
- Choose a light or dark theme, or follow the system, in **Settings** → **General**.
- Support Liquid Glass on design boxes, slots and component uses with `glass=regular`, `glass=clear`, `glass=none` and optional `interactive`.
- Add `OnElementMenu` and `OnNodeMenu` to `scenraview.Options`, telling the host app of right clicks on a design's elements and a flow chart's nodes, such as to open its own menus there.
- Add default visual, layout, multilingual, light/dark mode and render-check design guidelines to the bundled `scenra` and `scenra-app` skills so the assistant draws cleaner, adaptive interfaces by default.

## [0.3.0] - 2026-11-24

- Show designs in several languages: texts name messages of the folder's `locales/` catalogs, named as NSGo's, such as `en.yaml` and `zh-Hans.yaml`, with `key=`, and show them in the interface's language.
- Add `scenraview.Showcase`, drawing a project's components together as **Components** shows them, and draw flow charts with `scenraview.Still`, as the editor lays them out, both read-only, and pack a file's several designs in `scenraview.Still` under their names as the editor does instead of in one long column.
- Fix a chart holding a single flow taking no room, which kept **Fit** from framing it.
- Fix the names of designs and flows zoomed far out covering the row above.

## [0.2.0] - 2026-11-21

- Add `scenraview.Check`, telling the problems of a folder's or a file's Scenra files, as the MCP server reports them, without opening Scenra.
- Draw any design or view inside another with `use` by its name, leaving `components/` to the shared components.
- Shape a design's frame, its background, outline and corners, with `class=` on its line, so the assistant no longer draws a second frame inside it.
- Fix a view filling an outlet flashing at its device's size, a large rounded phone frame, before taking its outlet's, and framing its content with a device's outline and corners instead of its outlet's, which doubled a panel's or card's own.

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
