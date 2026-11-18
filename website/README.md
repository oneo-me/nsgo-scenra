# Scenra website

The site at <https://oneo-me.github.io/nsgo-scenra/>: the product page and a
privacy note in English and Simplified Chinese, in light and dark appearances.
It says what Scenra does and links the app's download to the latest GitHub
release. It follows the template of [Agio's site](https://github.com/oneo-me/nsgo-agio/tree/main/website),
built with Astro, Tailwind and TypeScript; the page ships no UI framework and
a small script for the appearance, language choice, copy buttons, tabs, the
screenshot lightbox and the mobile menu.

```sh
bun install
bun run dev          # http://localhost:4321/
bun run build        # typecheck, then build dist/
bun run preview      # serve dist/ at /nsgo-scenra/
bun run screenshots  # retake src/assets/screens/ from Scenra's demo build (macOS)
bun run og           # redraw public/og.png and public/og-zh.png (macOS)
```

## Words, language and appearance

All of the site's words are in `src/copy.ts`, with English and Chinese side
by side. Add both together; Chinese headings, titles and kickers omit the
trailing 。 and body sentences keep it. `src/site.ts` maps a language to its
path and holds the outside links. `Stage` takes a tone and features and notes
take an LED, named in `src/template.ts` and drawn in `src/styles/global.css`.

English is at the base and Chinese under `zh/`. A first visit from a Chinese
browser goes to `zh/`; a choice made with the switch is kept in `scenra-lang`.
The appearance follows the system until the switch chooses light, dark or
system, kept in `scenra-theme`. An inline head script applies both before the
first paint, so a stored choice does not flash the other one.

## Base path

The site is published under the repository's GitHub Pages project URL, so a
production build uses `/nsgo-scenra/`; the dev server uses `/`. A custom domain
sets both explicitly, along with the Umami script's `data-domains` in
`src/layouts/Layout.astro`:

```sh
SITE_URL=https://scenra.example BASE_PATH=/ bun run build
```

## Screenshots

`src/assets/screens/en/` and `zh/` hold the app in each language's words and
both appearances: `app` a desktop design, `phone` a phone design and `flow` a
flow chart, each beside the conversation that drew them, as `-light.png` and
`-dark.png`. They come from Scenra's demo build (`cmd/scenra/demo.go`), the
real app showing a scripted conversation without an agent CLI, open on the
music app in `scripts/music/`, whose `zh/` and `en/` are the same project in
each language. `bun run screenshots` builds the demo, runs it with a HOME of
its own, a fresh copy of the project and the language and appearance it
needs, and captures each window. It needs macOS and a screen; quit any demo
still open first, as the script captures the first window it finds. The
READMEs show `app` too.

The site imports the PNGs and lets Astro make AVIF and WebP versions at
several widths; CSS shows the image that matches the chosen appearance.

## Social cards

`bun run og` renders `public/og.png` (English) and `public/og-zh.png` (Chinese)
with headless Chrome from the `app` screenshots. `public/icon.png` and
`public/apple-touch-icon.png` are the app's icon. The generated files are
committed; a build needs neither.

## Statistics and deployment

The page counts visits in Scenra's Umami site, tagged `website`, only on the
domains listed in the script's `data-domains`; the privacy page says what is
counted.

The [Website workflow](../.github/workflows/website.yml) builds `website/dist`
and publishes it to GitHub Pages on pushes to `main` that touch `website/`.
