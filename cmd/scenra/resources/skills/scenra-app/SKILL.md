---
name: scenra-app
description: Work with the user in the Scenra app, which draws a folder's Scenra files beside your conversation: ask its scenra MCP server, naming your tab, what the user sees, selected and commented, and what the files you wrote got wrong. Use whenever your instructions say the user has a folder open in a Scenra tab, together with the scenra skill, which teaches the files themselves.
---

# Working in the Scenra app

The Scenra app opens each folder in a tab of its own, beside a home page of the folders opened lately. A tab lists the folder's Scenra files on the left, draws the one open in the middle and runs its conversations on the right; you are one of them, about that folder alone. Your instructions name the folder and the tab's ID. The scenra skill teaches the files; this one, how to see what the user sees.

## The tab

Give every tool of the `scenra` MCP server `tab`, the ID your instructions give, such as `tab="1f3a9c02"`. Other tabs show other folders, so a call naming none, or another's, fails rather than telling of a folder you are not working in; when one fails because the tab is gone, the user closed it: say so instead of guessing another.

## Working on files

1. Before you change files, call `get_view`. It names the open folder, and the folder holding its Scenra files when that is its `design/` folder, as a repository with no Scenra files of its own keeps them there; you still work in the open folder, so write files where it names. It names the file the user is looking at and what they selected, lists the comments and notes on elements and the problems of the folder's files. Work on what is selected when a request may be about it, such as "make this bigger". Once you have read the notes, pass `notes="open"` for only those of the open file and the components it uses, or `notes="none"`; the problems show in full either way.
2. Edit the files as the scenra skill says. The editor reloads the file shown as you write it and shows the files you make; the user cannot edit them while you work, and what you write in one turn undoes as one step.
3. After you write them, call `get_view` again: it reports the files that do not parse, the classes that draw nothing, the taps that lead nowhere and what lays out wrong, such as text cut off, elements outside their design or of no size, and `w-full` filling nothing, besides the problems the scenra skill names. Fix what it reports.
4. When a layout looks off, call `get_layout` with a file's path, and a design's ID or a component's variant, to see where each of its elements laid out, its x, y, width and height from the design's top left corner; give it `element=` a path, such as `0.2.1`, for only that element and what it holds, and `depth=` to list only that many levels under it, counting the rest. Paths inside a component follow a colon, and those of a design an outlet shows a bar.

`get_view` and `get_layout` name elements by their paths, as the scenra skill says, and a design of a file of several by its ID.

## What the user does

- The user selects an element or a node, which `get_view` tells, and leaves comments on elements for you, which the scenra skill says how to resolve.
- Sending a message saves what the user changed, so the files are what they see.
- The user plays designs in the editor, pressing their taps, so make the taps you write lead where the screens go.
- A file that never parsed shows its error, with a button asking you to fix it.
