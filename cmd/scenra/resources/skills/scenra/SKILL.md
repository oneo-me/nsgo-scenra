---
name: scenra
description: Design app screens and draw flow charts in Scenra files (.scenra), which the user sees drawn in the Scenra editor. Use when asked to design, draw or change an app's screens, interfaces or flows in a .scenra file.
---

# Scenra files

A Scenra file is plain text describing an app's screens: each screen has a design, a draft of how it looks, and flow charts of what happens on it, such as what the app does when the user presses its main button. The user sees the file drawn in the Scenra editor: the screens on the left, the open design or flow in the middle. When the user only wants a flow chart, draw flows and leave the design empty.

## Working on a file

1. Before you change the file, call the `get_view` tool of the `scenra` MCP server with the file's absolute path. It says what the user is looking at and has selected, lists their comments and the file's problems. Work on what is selected when a request may be about it, such as "make this bigger".
2. Read the file and edit it with your own file tools. Change only what the request needs, keep the IDs that exist and never invent screens or nodes the user did not ask for without saying so. The editor reloads the file as you write it, and the user cannot edit it while you work.
3. After you write it, call `get_view` again: it reports whether the file parses and the classes that draw nothing. Fix what it reports.
4. Say briefly what you changed. Keep labels short and put detail in notes. Write the interface's texts in the user's language.

The editor saves the file in its canonical form, which drops `#` comments, so do not keep notes in them.

When a design or flow is about code at hand, read the code first and draw what it actually does, naming the files behind a step in its note when that helps.

## Format

```
# A comment runs to the end of the line; indentation is free.
title "Shop"

screen signin "Sign in" device=phone
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

  flow login "Signing in"
    direction right
    node start "Tap Sign in" kind=start
    node check "Check password" kind=decision note="Lock after 5 tries"
    node home "Show home" kind=end status=done
    start -> check
    check -> home "valid"
    check ..> start "invalid"
    a -> b -> c
```

- `title` names the whole project, such as the app.
- `screen <id> "<title>"` starts a screen, with `device=` phone (390 by 800 points, the default), tablet (820 by 1080) or desktop (1200 by 760). Its interface comes before its flows: one element a line, and a `box` holds the lines up to its `end`.
- `flow <id> "<title>"` starts a flow chart of the screen above it. `direction down` lays it out top to bottom, `direction right` left to right.
- `node <id> "<label>"` is a step, with optional `note="…"`, `kind=` task (the default), start, end, decision, data or note, and `status=` todo, doing, done, blocked or skipped, when the chart tracks progress; skipped fades the step.
- `a -> b "label"` connects nodes, `a ..> b` dashed for optional or indirect links, and `a -> b -> c` chains them. An edge may name nodes no line declares; they take their ID as their label.
- IDs are letters, digits, `_`, `-` or `.`. Screen and flow IDs are unique in the file.
- A file without screens is a single flow, which reads as screen `main` with flow `main`.

Use `kind=start` and `kind=end` for entry and exit points and `kind=decision` for branches, with the answers as edge labels.

## Designs

A design is drawn with a few primitives, an element a line, each styled with Tailwind CSS 4 classes in `class="…"`:

- `box`: a rectangle laying out what it holds, a column unless its class has flex-row: with classes, any bar, panel, card, button, field, pill, circle or line
- `text`: its text, which wraps; size, weight and color it with classes
- `icon`: the Lucide icon its text names, as high as its font size: size it with size-* and color it with text-*
- `image`: a picture its text describes: the image at the URL `src="…"`, or a placeholder; size it with classes
- `svg`: any drawing, such as a chart, a logo or an illustration: its text is an SVG document, quoted with single quotes inside, sized by its class or else its viewBox

Draw any interface with them, as a designer draws in Figma: buttons, fields, switches, tabs, cards, lists, charts and navigation are boxes, texts, icons and SVG shaped by classes, with exactly the look the app needs. Make designs look like a finished product, with a clear hierarchy, consistent spacing and realistic content, not wireframes.

Layout is flexbox: a box is a column unless flex-row, children stretch across it, flex-1 fills the rest, and absolute with top-*, left-*, right-* or bottom-* places a box over its parent, as a badge or a floating button. Classes cover sizes (w-*, h-*, size-*, w-full, w-1/2), spacing (p-*, gap-*, m-*), colors (bg-*, text-*, border-*, with /opacity such as bg-black/40, and any value as bg-[#0f172a]), the palette (zinc, slate, blue, amber, emerald, rose and the others, 50 to 950), corners (rounded-*, rounded-full), borders, shadows (shadow-sm to shadow-2xl), gradients (bg-linear-to-br from-indigo-500 to-pink-500), opacity, overflow-hidden, text (text-xs to text-6xl, font-medium, font-bold, leading-*, tracking-*, text-center, truncate, line-clamp-2, uppercase) and theme colors that follow light and dark mode (bg-background, text-foreground, bg-card, bg-primary, text-primary-foreground, bg-muted, text-muted-foreground, border-border).

The screen is the device's size and clips what overflows; give the main box flex-1 to fill it, and draw the status bar, navigation bars and tab bars yourself where the app has them.

An icon takes the name of any Lucide icon (lucide.dev), by its current name or an older one such as home or alert-circle; common ones are house, search, bell, settings, user, users, circle-user, mail, message-circle, message-square, phone, send, inbox, heart, star, bookmark, share-2, link, paperclip, image, camera, video, mic, music, play, pause, calendar, clock, map-pin, navigation, compass, globe, shopping-cart, shopping-bag, store, credit-card, wallet, gift, tag, package, truck, file, file-text, folder, download, upload, cloud, plus, minus, x, check, pencil, trash, copy, funnel, sliders-horizontal, list, layout-grid, layout-dashboard, menu, ellipsis, ellipsis-vertical, chevron-right, chevron-left, chevron-down, chevron-up, arrow-right, arrow-left, arrow-up, arrow-down, external-link, refresh-cw, rotate-ccw, log-in, log-out, lock, key, shield, eye, eye-off, info, circle-alert, triangle-alert, circle-check, circle-x, circle-question-mark, chart-bar, chart-line, chart-pie, activity, trending-up, trending-down, zap, sparkles, flame, award, trophy, target, sun, moon, wifi, battery, smartphone, monitor, qr-code, scan, book, briefcase, building, coffee, utensils, car, plane, bike, dumbbell, headphones, graduation-cap. Draw a logo, a chart or an illustration with svg.

`get_view` names elements by their paths: an element's index among its siblings at each level from 0, joined by dots, so 1.0 is the first element the second top-level element holds.

## Comments

The user comments on elements to ask for changes: an element's `comment="…"` is the user's, not part of the design. Do what a comment asks, or discuss it when it is unclear, then remove the attribute and say what you did. Keep comments on the elements you rewrite until they are resolved, and never add one yourself.
