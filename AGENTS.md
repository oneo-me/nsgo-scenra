# Scenra

A desktop app and flow designer built on [NSGo](https://github.com/nsgo-dev/nsgo): a Scenra file's screens and their flow charts on the left, an AI assistant that draws them on the right. `scenraview` gives other apps the same editor; `cmd/scenra` is the app. Read [ARCHITECTURE.md](ARCHITECTURE.md) before coding.

## Principles

- Follow NSGo's conventions: user-visible words go through `scenraview/locales/`, or `cmd/scenra/locales/` for the app's own, in both languages, never string literals.
- Keep `scenraview` usable by other apps: what is Scenra's alone, such as its menu, settings and statistics, stays in `cmd/scenra`.
- Tests run against a scripted Agio driver; never start a real agent CLI from tests.
- Never let usage statistics carry document content, messages or paths, or report from `nsgo dev` builds and tests.

## Verification

- Run `go tool nsgo check` and `go tool nsgo test`, never `go vet`, `go test` or `go build` directly.
- Build the website from `website/` with `bun run build`; the Website workflow publishes it to GitHub Pages.
- Do not run `go mod tidy`: it drops the requirements `nsgo check` needs for TSX type generation. Update NSGo with `go get github.com/nsgo-dev/nsgo@<version>`.

## Releasing

- Release by pushing a `v<version>` tag, such as `v0.1.0`, once `CHANGELOG.md` has that version's section; the Release workflow builds on a Mac and publishes it.
- The updates' secret key lives only in the `NSGO_UPDATES_KEY` Actions secret; never commit it.

## Documentation

- Update a document only when a fact its reader needs changes.
- Avoid secondary labels and descriptions that restate a heading or value; let the layout explain.
- Write only what code and comments cannot show: purpose, boundaries and reasons, not names, fields, defaults or step-by-step behavior.
- Keep each document within its budget; check its size after editing it and trim it before handing off when it is over.
- Record completed user-visible outcomes as one-clause `Unreleased` bullets in [CHANGELOG.md](CHANGELOG.md), ordered by type and merging related ones.
- Repair documents you touch instead of only appending to them.
- Keep `README.md` and `README.zh-CN.md` equivalent; everything else, including code comments, is English.
- Keep the website's English and Chinese words together in `website/src/copy.ts`.
