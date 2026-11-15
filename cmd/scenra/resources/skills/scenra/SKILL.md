---
name: scenra
description: Design app screens and draw flow charts in Scenra files (.scenra), which the user sees drawn in the Scenra editor. Use when asked to design, draw or change an app's screens, interfaces or flows in a .scenra file.
---

# Scenra files

Scenra files (`.scenra`) are plain text, and each holds one thing: a design, a draft of how one screen of an app looks, or a flow chart, such as what the app does when the user presses a screen's main button. The user opens a folder in the Scenra editor, which lists the folder's Scenra files as a tree on the left and draws the one open in the middle. A project is a folder of them: give each screen a design file named after it, such as `sign-in.scenra`, and put what happens on that screen in its flow beside it, named the same with `-flow`, such as `sign-in-flow.scenra`; the editor keeps the two together and switches between them. Group screens in folders as the app grows. When the user only wants a flow chart, make only a flow file, still named `<name>-flow.scenra`. Never name a design with `-flow` at the end. Put the pieces several screens share, such as a tab bar, a navigation bar or a card, in components, which live in one folder, `components/`, at the top of the open folder.

## Working on files

1. Before you change files, call the `get_view` tool of the `scenra` MCP server. It names the open folder and the file the user is looking at and what they selected, lists the comments and notes on elements and the problems of the folder's files. Work on what is selected when a request may be about it, such as "make this bigger".
2. Read the files and edit them, or make new ones in the folder, with your own file tools. Change only what the request needs, keep the IDs and files that exist and never invent screens or nodes the user did not ask for without saying so. The editor reloads the file shown as you write it, shows the files you make, and the user cannot edit them while you work.
3. After you write them, call `get_view` again: it reports the files that do not parse, the classes that draw nothing and the taps that lead nowhere. Fix what it reports.
4. Say briefly what you changed. Keep labels short and put detail in notes. Write the interface's texts in the user's language.

The editor saves files in their canonical form, which drops `#` comments, so do not keep notes in them.

When a design or flow is about code at hand, read the code first and draw what it actually does, naming the files behind a step in its note when that helps.

## Format

A file's first line says what it holds: `design`, with `device=` phone (390 by 800 points, the default), tablet (820 by 1080) or desktop (1200 by 760), and optional `width=` and `height=` in points, 100 to 4000, which replace the device's size where the screen needs another, such as `design device=desktop width=1440 height=900` or a long page; `component`, a piece designs share, with the same `device=`, `width=` and `height=`, or `flow`, with `direction=down` (the default) to lay it out top to bottom or `direction=right` left to right. Any first line may take `name="…"`, the name the editor shows for the file instead of its file name, such as `design name="Sign in" device=phone`; give one when a readable title in the user's language says more than the file name, which stays short and in English. A file holds one design, component or flow, never two.

```
# A comment runs to the end of the line; indentation is free.
design device=phone
box class="flex-1 gap-4 bg-zinc-50 p-6 pt-16"
  text "Welcome back" class="text-3xl font-bold text-zinc-900"
  text "Sign in to order ahead." class="text-zinc-500"
  box class="h-12 flex-row items-center gap-3 rounded-xl border border-zinc-200 bg-white px-4"
    icon mail class="size-4 text-zinc-400"
    text "you@example.com" class="text-zinc-400"
  end
  box class="h-12 items-center justify-center rounded-xl bg-amber-600 shadow-md"
    text "Sign in" class="font-semibold text-white"
  end
  image "Latte art seen from above" class="h-40 rounded-2xl"
  svg "<svg viewBox='0 0 120 40'><path d='M0 32 L30 20 L60 26 L90 8 L120 14' stroke='#d97706' stroke-width='3' fill='none'/></svg>" class="h-16 w-full"
end
```

```
flow direction=right
node start "Tap Sign in" kind=start
node check "Check password" kind=decision note="Lock after 5 tries"
node home "Show home" kind=end status=done
start -> check
check -> home "valid"
check ..> start "invalid"
a -> b -> c
```

- A design is one element a line, and a `box` holds the lines up to its `end`.
- `node <id> "<label>"` is a step of a flow, with optional `note="…"`, `kind=` task (the default), start, end, decision, data or note, and `status=` todo, doing, done, blocked or skipped, when the chart tracks progress; skipped fades the step.
- `a -> b "label"` connects nodes, `a ..> b` dashed for optional or indirect links, and `a -> b -> c` chains them. An edge may name nodes no line declares; they take their ID as their label.
- Node IDs are letters, digits, `_`, `-` or `.`, unique in their flow.
- The file's name is its title: there is no title line.

Use `kind=start` and `kind=end` for entry and exit points and `kind=decision` for branches, with the answers as edge labels.

## Designs

A design is drawn with a few primitives, an element a line, each styled with Tailwind CSS 4 classes in `class="…"`:

- `box`: a rectangle laying out what it holds, a column unless its class has flex-row: with classes, any bar, panel, card, button, field, pill, circle or line
- `text`: its text, which wraps; size, weight and color it with classes
- `icon`: the Lucide icon its text names, as high as its font size: size it with size-* and color it with text-*
- `image`: a picture its text describes: the image at the URL `src="…"`, or a placeholder; size it with classes
- `svg`: any drawing, such as a chart, a logo or an illustration: its text is an SVG document, quoted with single quotes inside, sized by its class or else its viewBox
- `use`: the component its text names, a file of `components/`, drawn as a box with its classes holding the component's elements; see Components

Draw any interface with them, as a designer draws in Figma: buttons, fields, switches, tabs, cards, lists, charts and navigation are boxes, texts, icons and SVG shaped by classes, with exactly the look the app needs. Make designs look like a finished product, with a clear hierarchy, consistent spacing and realistic content, not wireframes.

Layout is flexbox: a box is a column unless flex-row, children stretch across it, flex-1 fills the rest, and absolute with top-*, left-*, right-* or bottom-* places a box over its parent, as a badge or a floating button. Classes cover sizes (w-*, h-*, size-*, w-full, w-1/2), spacing (p-*, gap-*, m-*), colors (bg-*, text-*, border-*, with /opacity such as bg-black/40, and any value as bg-[#0f172a]), the palette (zinc, slate, blue, amber, emerald, rose and the others, 50 to 950), corners (rounded-*, rounded-full), borders, shadows (shadow-sm to shadow-2xl), gradients (bg-linear-to-br from-indigo-500 to-pink-500), opacity, overflow-hidden, text (text-xs to text-6xl, font-medium, font-bold, leading-*, tracking-*, text-center, truncate, line-clamp-2, uppercase) and theme colors that follow light and dark mode (bg-background, text-foreground, bg-card, bg-primary, text-primary-foreground, bg-muted, text-muted-foreground, border-border).

The design is its size, the device's or its own, and clips what overflows; change `width=` or `height=` when the content needs room or the user asks for a screen size; give the main box flex-1 to fill it, and draw the status bar, navigation bars and tab bars yourself where the app has them.

An icon takes the name of any Lucide icon (lucide.dev), by its current name or an older one such as home or alert-circle; common ones are house, search, bell, settings, user, users, circle-user, mail, message-circle, message-square, phone, send, inbox, heart, star, bookmark, share-2, link, paperclip, image, camera, video, mic, music, play, pause, calendar, clock, map-pin, navigation, compass, globe, shopping-cart, shopping-bag, store, credit-card, wallet, gift, tag, package, truck, file, file-text, folder, download, upload, cloud, plus, minus, x, check, pencil, trash, copy, funnel, sliders-horizontal, list, layout-grid, layout-dashboard, menu, ellipsis, ellipsis-vertical, chevron-right, chevron-left, chevron-down, chevron-up, arrow-right, arrow-left, arrow-up, arrow-down, external-link, refresh-cw, rotate-ccw, log-in, log-out, lock, key, shield, eye, eye-off, info, circle-alert, triangle-alert, circle-check, circle-x, circle-question-mark, chart-bar, chart-line, chart-pie, activity, trending-up, trending-down, zap, sparkles, flame, award, trophy, target, sun, moon, wifi, battery, smartphone, monitor, qr-code, scan, book, briefcase, building, coffee, utensils, car, plane, bike, dumbbell, headphones, graduation-cap. Draw a logo, a chart or an illustration with svg.

`get_view` names elements by their paths: an element's index among its siblings at each level from 0, joined by dots, so 1.0 is the first element the second top-level element holds.

## Components

A component is a piece of interface several designs or components share, drawn once in a file of `components/` at the top of the open folder, so changing it changes every screen using it. Its first line is `component`; the rest is elements, as in a design, drawn at its device's width, or its `width=`, and as high as they need unless it sets `height=`. `use <name>` draws the component named by its path in `components/` without `.scenra`: `use tab-bar` draws `components/tab-bar.scenra`, and `use nav/item` draws `components/nav/item.scenra`.

```
# components/tab-bar.scenra
component
box class="flex-row border-t border-zinc-200 bg-white px-2 pt-2 pb-7"
  box class="flex-1 items-center gap-1"
    icon house class="size-6 text-amber-600"
    text "Home" class="text-[10px] font-medium text-amber-600"
  end
end
```

```
# menu.scenra
design
box class="flex-1 bg-zinc-50"
  text "Menu" class="p-5 text-2xl font-bold"
  use tab-bar class="mt-auto"
end
```

- A use is one element: a box with its classes holding the component's elements, so place and size it with classes as any box. It holds nothing itself and takes no end.
- Make a component of a piece the app's screens repeat, such as their tab bar, or when the user asks for one.
- Components may use other components, never themselves, directly or through others.
- Write a component as a whole, the same on every screen. A tab bar marks the screen shown by itself when its items open the screens, as Interactions shows; give what else differs to the designs, or make one component for each version.
- `get_view` lists the components there are and reports uses of missing ones and loops.

## Interactions

The editor plays a design as the app would run: pressing an element with `tap="…"`, or one inside it, does what the tap says, so the user can click through the screens.

- `tap="open <design>"` opens another design, named by its path in the open folder without `.scenra`, such as `tap="open menu"` or `tap="open orders/detail"`.
- `tap="close"` closes the screen shown, returning to the design that opened it, as a back button, a dialog's Cancel or a sheet's close button.
- `tap="tab <name>"` shows the elements with `panel=<name>` in place of the panels beside them. The first panel among its siblings shows until a tab shows another, so draw the panels of one place side by side, as the content of a segmented control or of tabs.

A box whose tap leads to what shows, opening the design shown or showing its panel, is selected: style it with `selected:` classes, and what it holds with `group-selected:` classes and `group` on the box.

```
box class="flex-row gap-2"
  box class="group rounded-full px-4 py-2 selected:bg-zinc-900" tap="tab coffee"
    text "Coffee" class="text-zinc-700 group-selected:text-white"
  end
  box class="group rounded-full px-4 py-2 selected:bg-zinc-900" tap="tab tea"
    text "Tea" class="text-zinc-700 group-selected:text-white"
  end
end
box class="gap-2" panel=coffee
  text "Latte"
end
box class="gap-2" panel=tea
  text "Matcha"
end
```

In a tab bar component, give each item `tap="open <screen>"` and the selected styles, so every screen using it marks its own tab. Add taps where the user asks for a clickable prototype or where they make the flow between screens clear; open only designs that exist or that you make.

## Comments and notes

The user comments on elements to ask for changes: an element's `comment="…"` is the user's, not part of the design. Do what a comment asks, or discuss it when it is unclear, then remove the attribute and say what you did. Keep comments on the elements you rewrite until they are resolved, and never add one yourself.

An element's `note="…"` says what the design cannot show, such as what it does, where its data comes from or when it changes. Notes stay: read them as part of the design, keep them true as you change it, and add one when the user asks or when a behavior matters and the design cannot show it.
