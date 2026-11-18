// Retakes the website's screenshots from Scenra's demo build on macOS: the
// real app showing a scripted conversation (cmd/scenra/demo.go) beside the
// music app designed in scripts/music, so the images need no agent CLI.
// Each language and appearance gets a desktop design, a flow chart and a
// phone design. Run `bun run screenshots` after changing the app's
// look.
import { execFileSync, spawn } from 'node:child_process';
import { cpSync, mkdirSync, mkdtempSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'darwin') {
  console.error('The screenshots need macOS: they capture a native NSGo window.');
  process.exit(1);
}

const scripts = path.dirname(fileURLToPath(import.meta.url));
const website = path.resolve(scripts, '..');
const repo = path.resolve(website, '..');
const screens = path.join(website, 'src', 'assets', 'screens');
// Outside the repository, so the sidebar shows no Git changes.
const scratch = mkdtempSync(path.join(os.tmpdir(), 'scenra-screenshots-'));
const binary = path.join(scratch, 'scenra-demo');

// The files each shot opens, in turn, from the project's folder: a flow
// chart draws only once a design showed before it.
const shots = {
  app: { open: ['desktop-discover.scenra', 'desktop-now-playing.scenra'] },
  flow: { open: ['desktop-discover.scenra', 'flows/desktop-home-to-player.scenra'] },
  phone: { open: ['mobile-now-playing.scenra', 'mobile-discover.scenra'] },
};
const locales = { en: ['(en)', 'en_US'], zh: ['(zh-Hans)', 'zh_CN'] };

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

function windowID() {
  try {
    return execFileSync('xcrun', ['swift', path.join(scripts, 'window-id.swift'), 'scenra-demo'], {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'ignore'],
    }).trim();
  } catch {
    return '';
  }
}

async function capture({ lang, appearance, shot }) {
  // A HOME of the demo's own, so the user's tabs, conversations and
  // settings are neither read nor changed, and a fresh copy of the
  // project, which the app saves in its canonical form.
  const home = path.join(scratch, 'home');
  const project = path.join(scratch, 'music');
  rmSync(home, { recursive: true, force: true });
  rmSync(project, { recursive: true, force: true });
  mkdirSync(home, { recursive: true });
  cpSync(path.join(scripts, 'music', lang), project, { recursive: true });

  const { open } = shots[shot];
  const env = {
    ...process.env,
    HOME: home,
    SCENRA_DEMO_OPEN: open.map((file) => path.join(project, file)).join(':'),
    SCENRA_DEMO_APPEARANCE: appearance,
    SCENRA_DEMO_LANG: lang,
  };
  const [languages, locale] = locales[lang];
  const output = path.join(screens, lang, `${shot}-${appearance}.png`);
  const child = spawn(binary, ['-AppleLanguages', languages, '-AppleLocale', locale], { cwd: repo, env, stdio: 'ignore' });

  try {
    let id = '';
    for (let attempt = 0; attempt < 80 && !id; attempt++) {
      await sleep(100);
      if (child.exitCode !== null) throw new Error('The demo exited before its window opened.');
      id = windowID();
    }
    if (!id) throw new Error('The demo window did not appear.');
    // The demo opens a file a second.
    await sleep(1500 + open.length * 1000);

    mkdirSync(path.dirname(output), { recursive: true });
    execFileSync('screencapture', ['-x', '-o', `-l${id}`, output], { stdio: 'inherit' });
    console.log(`wrote ${path.relative(website, output)}`);
  } finally {
    child.kill('SIGTERM');
    await Promise.race([new Promise((resolve) => child.once('exit', resolve)), sleep(2000)]);
  }
}

try {
  console.log('Building the demo…');
  execFileSync('go', ['build', '-tags', 'scenrademo', '-o', binary, './cmd/scenra'], { cwd: repo, stdio: 'inherit' });
  for (const lang of Object.keys(locales)) {
    for (const appearance of ['light', 'dark']) {
      for (const shot of Object.keys(shots)) {
        await capture({ lang, appearance, shot });
      }
    }
  }
} finally {
  rmSync(scratch, { recursive: true, force: true });
}
