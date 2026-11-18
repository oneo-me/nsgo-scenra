// The site's languages, pages and outside links. Everything else is
// written in src/copy.ts, so the page is one document to read and one to
// render.

export type Lang = 'en' | 'zh';

export const langs: Lang[] = ['en', 'zh'];

export const langTag = (lang: Lang): string => (lang === 'zh' ? 'zh-CN' : 'en');

const repo = 'https://github.com/oneo-me/nsgo-scenra';

export const site = {
  name: 'Scenra',
  repo,
  readme: `${repo}#readme`,
  readmeZh: `${repo}/blob/main/README.zh-CN.md`,
  examples: `${repo}/tree/main/examples`,
  releases: `${repo}/releases/latest`,
  format: `${repo}/tree/main/flowfile`,
  changelog: `${repo}/blob/main/CHANGELOG.md`,
  license: `${repo}/blob/main/LICENSE`,
  issues: `${repo}/issues`,
  install: {
    unix: `curl -fsSL ${repo}/releases/latest/download/install.sh | sh`,
    windows: `irm ${repo}/releases/latest/download/install.ps1 | iex`,
  },
  library: 'go get github.com/oneo-me/nsgo-scenra',
  agio: 'https://oneo-me.github.io/nsgo-agio/',
  nsgo: 'https://nsgo.dev',
  nsgoRepo: 'https://github.com/nsgo-dev/nsgo',
  blog: 'https://oneo.me',
  x: 'https://x.com/oneo666',
  author: 'oneo',
} as const;

/** An internal path below the GitHub Pages base, with a leading slash. */
export function path(p = ''): string {
  return `${import.meta.env.BASE_URL}${p}`.replace(/\/{2,}/g, '/');
}

/** The same path in a language: English at the base, Chinese under zh/. */
export function pagePath(lang: Lang, p = ''): string {
  return path(`${lang === 'zh' ? 'zh/' : ''}${p}`);
}
