// Renders the 5:7 promo video from the site's screenshots: headless Chrome
// draws each frame of a page whose every style is a function of time, and
// ffmpeg encodes them with music.mjs's music. Its first frame is the cover.
// Run `bun run video` after retaking the screenshots; it writes
// video/scenra.mp4, which is not committed, and `--stills=0,5` writes those
// seconds as PNGs instead.
import { spawn } from 'node:child_process';
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { music } from './music.mjs';

const scripts = path.dirname(fileURLToPath(import.meta.url));
const website = path.resolve(scripts, '..');
const out = path.join(website, 'video');
const chrome = process.env.CHROME ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const mark = readFileSync(path.join(website, 'public', 'favicon.svg'), 'utf8');
const shot = (lang, name) => `file://${path.join(website, 'src', 'assets', 'screens', lang, `${name}.png`)}`;

const W = 1080;
const H = 1512;
const FPS = 30;
const DURATION = 34;

const font =
  "-apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Segoe UI', 'PingFang SC', 'Noto Sans SC', sans-serif";
const mono = "ui-monospace, 'SF Mono', 'JetBrains Mono', Menlo, monospace";
const github =
  '<svg viewBox="0 0 16 16" fill="currentColor"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>';

// The scenes, from their start to their end in seconds, which the music's
// whooshes and last chord follow.
const times = {
  cover: [0, 2.6],
  window: [2.6, 7.8],
  draw: [7.8, 13.6],
  screens: [13.6, 18.2],
  flow: [18.2, 22.8],
  agents: [22.8, 26.8],
  open: [26.8, 31],
  outro: [31, 99],
};

// Each scene's words: an English headline over its Chinese line.
const scenes = {
  cover: ['Design apps.<br>By talking.', '把应用设计，说出来。'],
  window: ['Everything in one window.', '文件、画布与助手，同在一个窗口'],
  draw: ['Describe it. It draws it.', '说出想法，它来画'],
  screens: ['Screens that look real.', '像真应用一样的界面，点按即可试玩'],
  flow: ['Flows, charted for you.', '流程图，也帮你画好'],
  agents: ['Bring your own agent.', '用你自己的编程智能体'],
  open: ['Open source. Open format.', '开源、开放格式，可作为组件嵌入你的应用'],
};

const prompt = 'Design a now playing screen: big cover, synced lyrics, dark and calm.';
const agents = [
  ['Claude Code', '#e07a5f'],
  ['Codex', '#8b8ef0'],
  ['pi', '#45a56a'],
  ['CodeBuddy Code', '#e8a93a'],
];

// A component file, drawn by every design that uses it.
const scenra = `component name="Phone/Mini player"
box class="flex-row items-center gap-3 p-2"
  image "Cover" src="assets/cover.jpg"
  text "Blue in Green" name=title
  icon pause name=toggle-icon
end
variant paused toggle-icon=play`;

const embed = `e := scenraview.New(cx, scenraview.Options{})
e.OpenPath("designs")
return scenraview.Window{Editor: e}`;

const words = (html) =>
  html
    .split(/(<br>| )/)
    .map((w) => (w === '<br>' || w === ' ' ? w : `<span class="w">${w}</span>`))
    .join('');
const head = (id, top) =>
  `<div class="head" id="${id}Head" style="top:${top}px"><h2>${words(scenes[id][0])}</h2><div class="zh">${scenes[id][1]}</div></div>`;
const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;');

function page() {
  return `<!doctype html><html><head><meta charset="utf-8"><style>
  * { box-sizing: border-box; }
  html, body { margin: 0; width: ${W}px; height: ${H}px; overflow: hidden; }
  body { background: #07080b; color: #f1f3f7; font-family: ${font}; -webkit-font-smoothing: antialiased; }
  .blob { position: absolute; border-radius: 50%; }
  #b1 { width: 1300px; height: 1300px; left: -300px; top: -400px; background: radial-gradient(closest-side, rgb(91 94 224 / 34%), transparent 72%); }
  #b2 { width: 1100px; height: 1100px; left: 300px; top: 700px; background: radial-gradient(closest-side, rgb(69 165 106 / 16%), transparent 72%); }
  #b3 { width: 900px; height: 900px; left: 400px; top: 100px; background: radial-gradient(closest-side, rgb(224 91 160 / 12%), transparent 72%); }
  #grid { position: absolute; inset: -80px; background-image: radial-gradient(rgb(255 255 255 / 9%) 1.4px, transparent 1.6px);
    background-size: 40px 40px; -webkit-mask-image: radial-gradient(ellipse at 50% 45%, #000 20%, transparent 75%); }
  .scene { position: absolute; inset: 0; display: none; }
  .head { position: absolute; left: 70px; right: 70px; }
  h2 { margin: 0; font-size: 70px; font-weight: 750; line-height: 1.08; letter-spacing: -.045em; }
  .w { display: inline-block; }
  .zh { margin-top: 18px; font-size: 34px; color: #b9bbff; font-weight: 500; letter-spacing: .02em; }
  .card { position: absolute; border-radius: 26px; background: #16181e; border: 1px solid #2c313c;
    box-shadow: 0 50px 120px -40px #000, inset 0 1px 0 rgb(255 255 255 / 5%); overflow: hidden; }
  .view { position: absolute; overflow: hidden; border-radius: 22px; border: 1px solid #343a47; background: #1e1e1e;
    box-shadow: 0 60px 140px -40px #000, 0 0 80px -20px rgb(91 94 224 / 35%); }
  .stage { position: absolute; left: 0; top: 0; width: 2880px; height: 1738px; transform-origin: 0 0; }
  .stage img { position: absolute; inset: 0; width: 2880px; height: 1738px; }
  .chip { display: inline-flex; align-items: center; gap: 12px; padding: 14px 26px; border-radius: 999px;
    background: #15171d; border: 1px solid #2c313c; font-size: 28px; font-weight: 600; color: #d6d8e0; }
  .chip small { font-size: 24px; color: #9a9db0; font-weight: 500; }
  .dot { width: 14px; height: 14px; border-radius: 50%; }
  code { font-family: ${mono}; }

  /* Cover */
  #brand { position: absolute; left: 0; right: 0; top: 96px; display: flex; justify-content: center; align-items: center; gap: 22px; }
  #brand svg { width: 92px; height: 92px; filter: drop-shadow(0 16px 40px rgb(91 94 224 / 55%)); }
  #brand span { font-size: 64px; font-weight: 750; letter-spacing: -.045em; }
  #coverHead { text-align: center; }
  #coverHead h2 { font-size: 112px; line-height: 1.02; letter-spacing: -.05em; }
  #coverHead h2 .w:nth-last-child(-n+2) { color: #c9caff; }
  #coverHead .zh { font-size: 44px; margin-top: 26px; color: #e6e7ff; letter-spacing: .06em; }
  #coverFoot { position: absolute; left: 0; right: 0; top: 1390px; display: flex; justify-content: center; gap: 14px; }
  #coverFoot .chip { font-size: 24px; padding: 10px 20px; }

  /* The window */
  #win { position: absolute; left: 40px; top: 0; width: 1000px; height: 604px; transform-origin: 50% 0; display: none; }
  #spot { position: absolute; border: 8px solid #a5a7ff; border-radius: 30px; box-shadow: 0 0 0 4000px rgb(5 6 10 / 62%), 0 0 60px #8b8ef0; }
  #parts { position: absolute; left: 0; right: 0; top: 1030px; display: flex; justify-content: center; gap: 16px; }
  #parts .chip.on { background: #5b5ee0; border-color: #8b8ef0; color: #fff; box-shadow: 0 10px 40px -8px #5b5ee0; }
  #parts .chip.on small { color: #e0e1ff; }
  #lang { position: absolute; left: 50%; top: 1030px; width: 520px; margin-left: -260px; height: 86px; border-radius: 999px;
    background: #15171d; border: 1px solid #2c313c; display: flex; align-items: center; font-size: 30px; font-weight: 650; }
  #lang span { flex: 1; text-align: center; position: relative; z-index: 1; }
  #knob { position: absolute; top: 8px; left: 8px; width: 252px; height: 68px; border-radius: 999px; background: #5b5ee0; box-shadow: 0 10px 40px -8px #5b5ee0; }
  #langNote { position: absolute; left: 0; right: 0; top: 1150px; text-align: center; color: #9a9db0; font-size: 26px; }

  /* Describe it */
  #chat { left: 70px; right: 70px; top: 330px; height: 430px; padding: 34px; }
  .bubble { margin-left: auto; max-width: 720px; background: #2a2b31; border-radius: 24px; padding: 22px 28px;
    font-size: 30px; line-height: 1.45; color: #ececf1; min-height: 86px; }
  .caret { display: inline-block; width: 3px; height: 34px; background: #c9caff; vertical-align: -6px; margin-left: 2px; }
  .tool { margin-top: 20px; display: flex; align-items: center; gap: 16px; font-size: 25px; color: #9aa1ad; }
  .tool i { width: 12px; height: 12px; border-radius: 50%; background: #45a56a; box-shadow: 0 0 14px #45a56a; }
  .tool code { color: #e6e7ee; font-size: 23px; background: #24262d; padding: 5px 12px; border-radius: 9px; }
  #think { margin-top: 30px; font-size: 26px; font-weight: 600;
    background: linear-gradient(90deg, #5d6170 0%, #f1f3f7 50%, #5d6170 100%); background-size: 200% 100%;
    -webkit-background-clip: text; background-clip: text; color: transparent; }
  #link { position: absolute; left: 538px; top: 762px; width: 4px; height: 56px;
    background: linear-gradient(#8b8ef0, #45a56a); border-radius: 4px; transform-origin: 50% 0; }
  #design { left: 40px; top: 820px; width: 1000px; height: 600px; }
  .tile { position: absolute; background: #1b1c22; border: 1px solid #2a2c35; }
  #scan { position: absolute; top: 0; bottom: 0; width: 160px;
    background: linear-gradient(90deg, transparent, rgb(165 167 255 / 45%), transparent); mix-blend-mode: screen; }

  /* Screens */
  #phone { left: 70px; top: 640px; width: 380px; height: 780px; border-radius: 34px; background: #fafafa; }
  #desk { left: 140px; top: 380px; width: 900px; height: 570px; }
  .ring { position: absolute; width: 120px; height: 120px; margin: -60px 0 0 -60px; border-radius: 50%; border: 5px solid #fff; }
  #tapLabel { position: absolute; padding: 10px 20px; border-radius: 14px; background: #fff; color: #111; font-size: 24px; font-weight: 650;
    box-shadow: 0 20px 50px -10px #000; white-space: nowrap; transform-origin: 0 0; }
  #devices { position: absolute; left: 500px; top: 1060px; display: flex; flex-direction: column; gap: 16px; align-items: flex-start; }

  /* Flow */
  #flowView { left: 40px; top: 330px; width: 1000px; height: 1100px; }
  #token { position: absolute; width: 46px; height: 46px; margin: -23px 0 0 -23px; border-radius: 50%; background: #a5a7ff;
    box-shadow: 0 0 30px 12px rgb(139 142 240 / 80%), 0 0 90px 30px rgb(139 142 240 / 40%); }

  /* Agents */
  #orbit { position: absolute; left: 0; top: 0; width: ${W}px; height: ${H}px; }
  #hub { position: absolute; left: 450px; top: 760px; width: 180px; height: 180px; }
  #hub svg { width: 100%; height: 100%; filter: drop-shadow(0 20px 60px rgb(91 94 224 / 70%)); }
  .agent { position: absolute; padding: 22px 32px; border-radius: 24px; background: #15171d; border: 1px solid #2c313c;
    font-size: 38px; font-weight: 700; letter-spacing: -.03em; display: flex; gap: 18px; align-items: center; white-space: nowrap;
    box-shadow: 0 30px 70px -30px #000; }
  .agent .dot { width: 18px; height: 18px; }
  #agentsFoot { position: absolute; left: 0; right: 0; top: 1330px; text-align: center; font-size: 30px; color: #c8cad2; }
  #agentsFoot div { margin-top: 10px; color: #9a9db0; font-size: 26px; }

  /* Open */
  .open { left: 60px; right: 60px; padding: 30px 36px; }
  .open .label { display: flex; align-items: center; gap: 16px; font-size: 32px; font-weight: 700; letter-spacing: -.02em; }
  .open .label svg { width: 40px; height: 40px; }
  .open .label small { margin-left: auto; font-size: 24px; color: #9a9db0; font-weight: 500; letter-spacing: 0; }
  .open pre { margin: 20px 0 0; font-family: ${mono}; font-size: 23px; line-height: 1.65; color: #c8cad2; white-space: pre; min-height: 76px; }
  .tag { display: inline-block; font-size: 22px; padding: 6px 14px; border-radius: 10px; background: #24262d; color: #c9caff; font-family: ${mono}; }
  .k { color: #9a9cff; font-weight: 600; } .s { color: #8fd3a8; } .a { color: #e8a93a; }
  #stars { display: flex; gap: 14px; margin-top: 22px; }

  /* Outro */
  #outroMark { position: absolute; left: 440px; top: 380px; width: 200px; height: 200px; }
  #outroMark svg { width: 100%; height: 100%; filter: drop-shadow(0 20px 60px rgb(91 94 224 / 70%)); }
  #outro > div:not(#outroMark):not(#outroCta) { position: absolute; left: 0; right: 0; text-align: center; }
  #outroName { top: 620px; font-size: 110px; font-weight: 750; letter-spacing: -.05em; }
  #outroLine { top: 770px; font-size: 46px; font-weight: 650; letter-spacing: -.03em; color: #c9caff; }
  #outroZh { top: 840px; font-size: 36px; color: #e6e7ff; letter-spacing: .06em; }
  #outroCta { position: absolute; left: 50%; top: 960px; padding: 24px 46px; border-radius: 999px;
    background: #f1f3f7; color: #0b0c10; font-size: 34px; font-weight: 700; white-space: nowrap; }
  #outroFine { top: 1100px; color: #9a9db0; font-size: 28px; }
  #outroUrl { top: 1150px; color: #c8cad2; font-family: ${mono}; font-size: 28px; }
</style></head><body>
<div class="blob" id="b1"></div><div class="blob" id="b2"></div><div class="blob" id="b3"></div><div id="grid"></div>

<div class="scene" id="cover">
  <div id="brand">${mark}<span>Scenra</span></div>
  ${head('cover', 250)}
  <div id="coverFoot">
    <span class="chip">Claude Code · Codex · pi · CodeBuddy</span>
    <span class="chip">MIT · macOS · Windows · Linux</span>
  </div>
</div>

<div class="scene" id="window">
  ${head('window', 110)}
  <div id="parts">
    <span class="chip">Files <small>文件</small></span>
    <span class="chip">Canvas <small>画布</small></span>
    <span class="chip">Assistant <small>助手</small></span>
  </div>
  <div id="lang"><div id="knob"></div><span>English</span><span>简体中文</span></div>
  <div id="langNote">Light &amp; dark · 浅色与深色</div>
</div>

<div class="view" id="win"><div class="stage" id="winStage">
  <img src="${shot('en', 'app-dark')}"><img id="winZh" src="${shot('zh', 'app-light')}">
  <div id="spot"></div>
</div></div>

<div class="scene" id="draw">
  ${head('draw', 110)}
  <div class="card" id="chat">
    <div class="bubble"><span id="typed"></span><span class="caret" id="caret"></span></div>
    <div id="think">Thinking…</div>
    <div class="tool"><i></i>Viewed <code>mobile-now-playing.scenra</code></div>
    <div class="tool"><i></i>Read <code>components/player-bar.scenra</code></div>
    <div class="tool"><i></i>Wrote <code>desktop-now-playing.scenra</code></div>
  </div>
  <div id="link"></div>
  <div class="view" id="design"><div class="stage" id="designStage"><img src="${shot('en', 'app-dark')}"></div>
    <div id="tiles"></div><div id="scan"></div></div>
</div>

<div class="scene" id="screens">
  ${head('screens', 110)}
  <div class="view" id="desk"><div class="stage" id="deskStage"><img src="${shot('en', 'app-dark')}"></div></div>
  <div class="view" id="phone"><div class="stage" id="phoneStage"><img src="${shot('en', 'phone-light')}"></div></div>
  <div class="ring" id="ring1"></div><div class="ring" id="ring2"></div>
  <div id="tapLabel">Tap to play · 点按试玩</div>
  <div id="devices"><span class="chip">Phone <small>手机</small></span><span class="chip">Tablet <small>平板</small></span><span class="chip">Desktop <small>桌面</small></span></div>
</div>

<div class="scene" id="flow">
  ${head('flow', 110)}
  <div class="view" id="flowView"><div class="stage" id="flowStage"><img id="flowImg" src="${shot('en', 'flow-dark')}"><div id="token"></div></div></div>
</div>

<div class="scene" id="agents">
  ${head('agents', 110)}
  <svg id="orbit"></svg>
  <div id="hub">${mark}</div>
  ${agents.map(([name, color], i) => `<div class="agent"><span class="dot" style="background:${color};box-shadow:0 0 16px ${color}"></span>${name}</div>`).join('')}
  <div id="agentsFoot">Your sign-ins, your models, your folder.<div>你的账号、你的模型、你的项目</div></div>
</div>

<div class="scene" id="open">
  ${head('open', 110)}
  <div class="card open" id="open0" style="top:340px">
    <div class="label">${github} oneo-me/nsgo-scenra<small>MIT</small></div>
    <div id="stars"><span class="tag">★ Star</span><span class="tag">Issues</span><span class="tag">PRs welcome · 欢迎贡献</span></div>
  </div>
  <div class="card open" id="open1" style="top:570px">
    <div class="label"><span class="tag">.scenra</span> Plain text any agent writes<small>纯文本格式</small></div>
    <pre id="scenraCode"></pre>
  </div>
  <div class="card open" id="open2" style="top:1010px">
    <div class="label"><span class="tag">scenraview</span> Embed the editor<small>作为组件引用</small></div>
    <pre id="embedCode"></pre>
  </div>
</div>

<div class="scene" id="outro">
  <div id="outroMark">${mark}</div>
  <div id="outroName">Scenra</div>
  <div id="outroLine">Design apps. By talking.</div>
  <div id="outroZh">把应用设计，说出来。</div>
  <div id="outroCta">Download free · 免费下载</div>
  <div id="outroFine">Open source · macOS · Windows · Linux</div>
  <div id="outroUrl">github.com/oneo-me/nsgo-scenra</div>
</div>

<script>(${client.toString()})(${JSON.stringify({ times, prompt, scenra: esc(scenra), embed: esc(embed) })});</script>
</body></html>`;
}

// Runs in the page: defines render(t), which sets every style for second t.
function client({ times, prompt, scenra, embed }) {
  const $ = (id) => document.getElementById(id);
  const clamp = (x, a = 0, b = 1) => Math.min(b, Math.max(a, x));
  const ramp = (t, a, b) => clamp((t - a) / (b - a));
  const ease = (x) => (x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2);
  const out = (x) => 1 - Math.pow(1 - x, 3);
  // Overshoots a little before settling, for things that pop in.
  const back = (x) => 1 + 2.7 * Math.pow(x - 1, 3) + 1.7 * Math.pow(x - 1, 2);
  const lerp = (a, b, x) => a + (b - a) * x;
  const style = (el, s) => Object.assign(el.style, s);

  // Shows a scene and blurs it away over its last half second; returns the
  // seconds since it began, or -1 while it is hidden.
  function scene(id, t) {
    const [a, b] = times[id];
    const el = $(id);
    if (t < a || t >= b) {
      el.style.display = 'none';
      return -1;
    }
    const o = ease(ramp(t, b - 0.45, b));
    style(el, { display: 'block', opacity: 1 - o, transform: `translateY(${-70 * o}px) scale(${1 - 0.04 * o})`, filter: o ? `blur(${12 * o}px)` : 'none' });
    return t - a;
  }

  // Raises a headline word by word, its Chinese line after it.
  function headline(id, s) {
    const head = $(id + 'Head');
    const ws = head.querySelectorAll('.w');
    ws.forEach((w, i) => {
      const k = out(ramp(s, 0.05 + i * 0.07, 0.6 + i * 0.07));
      style(w, { opacity: k, transform: `translateY(${(1 - k) * 60}px) rotate(${(1 - k) * 4}deg)`, filter: `blur(${(1 - k) * 10}px)` });
    });
    const k = out(ramp(s, 0.3 + ws.length * 0.07, 0.9 + ws.length * 0.07));
    style(head.querySelector('.zh'), { opacity: k, transform: `translateY(${(1 - k) * 24}px)`, letterSpacing: `${0.02 + (1 - k) * 0.3}em` });
  }

  // Points a view at the image's rect from x, y, w wide.
  function camera(view, stage, x, y, w) {
    stage.style.transform = `scale(${view.clientWidth / w}) translate(${-x}px, ${-y}px)`;
  }

  // The spotlight's rects over the window, files, canvas and assistant, and
  // where it is s seconds into the window's scene, gliding between them.
  const parts = [[10, 92, 466, 1630], [490, 92, 1584, 1630], [2090, 92, 780, 1630]];
  function spotlight(s) {
    const at = clamp((s - 0.9) / 0.9, 0, 2);
    const i = Math.min(1, Math.floor(at));
    const g = ease(clamp((at - i - 0.55) / 0.45));
    return { rect: parts[i].map((v, j) => lerp(v, parts[i + 1][j], g)), on: i + (g > 0.5 ? 1 : 0) };
  }

  // The tiles hiding the design until it is drawn.
  const tiles = [];
  for (let r = 0; r < 6; r++)
    for (let c = 0; c < 10; c++) {
      const el = document.createElement('div');
      el.className = 'tile';
      style(el, { left: c * 100 + 'px', top: r * 100 + 'px', width: '101px', height: '101px' });
      $('tiles').appendChild(el);
      tiles.push([el, r, c]);
    }

  function highlight(s) {
    return s.replace(
      /("[^"\n]*")|\b(class|src|name)(?==)|^(\s*)(component|box|image|text|icon|end|variant|return)\b/gm,
      (all, str, attr, indent, word) =>
        str ? `<span class="s">${str}</span>` : attr ? `<span class="a">${attr}</span>` : `${indent}<span class="k">${word}</span>`,
    );
  }

  // The flow token's path through the chart, in the image's pixels.
  const route = [[978, 145], [978, 655], [967, 700], [967, 818], [956, 895], [800, 895], [798, 973], [884, 1010], [884, 1128]].map(
    ([x, y]) => [x * 1.44, y * 1.44],
  );
  const lengths = route.slice(1).map((p, i) => Math.hypot(p[0] - route[i][0], p[1] - route[i][1]));
  const total = lengths.reduce((a, b) => a + b, 0);
  function along(k) {
    let d = k * total;
    for (let i = 0; i < lengths.length; i++) {
      if (d <= lengths[i] || i === lengths.length - 1) {
        const f = clamp(d / lengths[i]);
        return [lerp(route[i][0], route[i + 1][0], f), lerp(route[i][1], route[i + 1][1], f)];
      }
      d -= lengths[i];
    }
  }

  window.render = (t) => {
    style($('b1'), { transform: `translate(${Math.sin(t * 0.3) * 120}px, ${Math.cos(t * 0.23) * 90}px)` });
    style($('b2'), { transform: `translate(${Math.cos(t * 0.27) * 140}px, ${Math.sin(t * 0.21) * 110}px)` });
    style($('b3'), { transform: `translate(${Math.sin(t * 0.19 + 2) * 160}px, ${Math.cos(t * 0.31) * 120}px)` });
    style($('grid'), { transform: `translate(${(t * 12) % 40}px, ${(t * 8) % 40}px)` });

    // The cover is whole at the first frame and only drifts until it leaves.
    let s = scene('cover', t);
    if (s >= 0) {
      const lift = Math.sin(s * 1.6) * 6;
      $('brand').style.transform = `translateY(${lift * 0.5}px)`;
      $('coverHead').style.transform = `translateY(${lift}px)`;
      $('coverFoot').style.opacity = 1 - ramp(s, 1.9, 2.3);
    }

    // The window rises from the cover's tilted card to face the viewer,
    // then the spotlight names its parts and the language switches.
    const win = $('win');
    if (t < times.window[1]) {
      const m = ease(ramp(t, 2.0, 3.1));
      const o = ease(ramp(t, times.window[1] - 0.45, times.window[1]));
      const float = Math.sin(t * 1.4) * 8 * (1 - m);
      style(win, {
        display: 'block',
        top: lerp(780, 380, m) + float - 70 * o + 'px',
        transform: `perspective(2000px) rotateX(${lerp(22, 0, m)}deg) rotateZ(${lerp(-2.5, 0, m)}deg) scale(${lerp(0.95, 1, m) - 0.04 * o})`,
        opacity: 1 - o,
        filter: o ? `blur(${12 * o}px)` : 'none',
      });
      camera(win, $('winStage'), 0, 0, 2880);
      const w = t - times.window[0];
      const { rect } = spotlight(w);
      const spot = clamp(ramp(w, 0.8, 1.1) - ramp(w, 3.3, 3.6));
      style($('spot'), { left: rect[0] + 'px', top: rect[1] + 'px', width: rect[2] + 'px', height: rect[3] + 'px', opacity: spot });
      $('winZh').style.opacity = ease(ramp(w, 3.8, 4.3));
    } else win.style.display = 'none';

    s = scene('window', t);
    if (s >= 0) {
      headline('window', s);
      const { on } = spotlight(s);
      $('parts').querySelectorAll('.chip').forEach((chip, i) => {
        const k = back(ramp(s, 0.7 + i * 0.12, 1.2 + i * 0.12));
        const lit = i === on && s > 0.9 && s < 3.4;
        chip.classList.toggle('on', lit);
        style(chip, { opacity: clamp(k), transform: `translateY(${(1 - k) * 40}px) scale(${lit ? 1.08 : 1})` });
      });
      $('parts').style.opacity = 1 - ramp(s, 3.3, 3.6);
      const l = back(ramp(s, 3.5, 3.9));
      style($('lang'), { opacity: clamp(l), transform: `scale(${0.7 + 0.3 * l})` });
      $('knob').style.transform = `translateX(${ease(ramp(s, 3.8, 4.3)) * 252}px)`;
      $('langNote').style.opacity = ramp(s, 4.1, 4.5);
    }

    s = scene('draw', t);
    if (s >= 0) {
      headline('draw', s);
      const c = out(ramp(s, 0.2, 0.7));
      style($('chat'), { opacity: c, transform: `translateY(${(1 - c) * 80}px) scale(${0.96 + 0.04 * c})` });
      $('typed').textContent = prompt.slice(0, Math.floor(clamp(s - 0.6, 0, 9) * 48));
      const typedEnd = 0.6 + prompt.length / 48;
      $('caret').style.opacity = s < typedEnd ? 1 : Math.floor(s * 2.5) % 2;
      style($('think'), { opacity: ramp(s, typedEnd + 0.1, typedEnd + 0.3), backgroundPosition: `${200 - ((s * 140) % 200)}% 0` });
      document.querySelectorAll('#chat .tool').forEach((el, i) => {
        const k = back(ramp(s, typedEnd + 0.4 + i * 0.35, typedEnd + 0.75 + i * 0.35));
        style(el, { opacity: clamp(k), transform: `translateX(${(1 - k) * 50}px)` });
      });
      const drawn = typedEnd + 1.5;
      $('link').style.transform = `scaleY(${out(ramp(s, drawn - 0.3, drawn))})`;
      const d = out(ramp(s, drawn - 0.2, drawn + 0.3));
      style($('design'), { opacity: d, transform: `translateY(${(1 - d) * 60}px)` });
      // Tiles flip away along the diagonal as the design is drawn.
      for (const [el, r, col] of tiles) {
        const k = ease(ramp(s, drawn + 0.1 + (r + col) * 0.07, drawn + 0.45 + (r + col) * 0.07));
        style(el, { opacity: 1 - k, transform: `scale(${1 - 0.6 * k}) rotate(${k * 20}deg)` });
      }
      $('scan').style.left = lerp(-200, 1100, ramp(s, drawn + 0.1, drawn + 1.4)) + 'px';
      const z = ease(ramp(s, drawn + 1.2, 5.8));
      camera($('design'), $('designStage'), lerp(530, 600, z), lerp(420, 470, z), lerp(1500, 1250, z));
    }

    s = scene('screens', t);
    if (s >= 0) {
      headline('screens', s);
      const bob = (delay) => Math.sin((s + delay * 3) * 1.8) * 10;
      // The desktop design swings in from the right, the phone rises before it.
      const dk = back(ramp(s, 0.25, 1.05));
      style($('desk'), {
        opacity: clamp(dk * 2),
        transform: `translate(${(1 - dk) * 500}px, ${bob(0.9) * 0.6}px) perspective(1800px) rotateY(${lerp(-24, -6, clamp(dk)) + ease(ramp(s, 1.2, 4)) * 6}deg)`,
      });
      camera($('desk'), $('deskStage'), lerp(544, 580, ramp(s, 0, 4.5)), 443, lerp(1472, 1400, ramp(s, 0, 4.5)));
      const pk = back(ramp(s, 0.5, 1.3));
      style($('phone'), {
        opacity: clamp(pk * 2),
        transform: `translateY(${(1 - clamp(pk)) * 600 + bob(0.3)}px) perspective(1600px) rotateY(${(1 - ease(ramp(s, 1.2, 2.4))) * 14}deg) rotateZ(${(1 - pk) * -6}deg)`,
      });
      // The phone's crop: 907, 144 is its top left in the screenshot.
      camera($('phone'), $('phoneStage'), 907, 144, 745);
      // A tap on the phone's player bar.
      const tap = s - 2.2;
      [$('ring1'), $('ring2')].forEach((ring, i) => {
        const k = ramp(tap, i * 0.18, 0.7 + i * 0.18);
        style(ring, { left: '260px', top: 640 + 674 + bob(0.3) + 'px', opacity: k > 0 && k < 1 ? 1 - k : 0, transform: `scale(${0.3 + k * 1.4})` });
      });
      const lab = back(ramp(tap, 0.2, 0.6));
      style($('tapLabel'), { left: '470px', top: 1290 + bob(0.3) + 'px', opacity: clamp(lab), transform: `scale(${0.6 + 0.4 * lab})` });
      $('devices').querySelectorAll('.chip').forEach((chip, i) => {
        const k = back(ramp(s, 1.4 + i * 0.12, 1.8 + i * 0.12));
        style(chip, { opacity: clamp(k), transform: `translateX(${(1 - k) * 80}px)` });
      });
    }

    s = scene('flow', t);
    if (s >= 0) {
      headline('flow', s);
      const v = out(ramp(s, 0.2, 0.7));
      style($('flowView'), { opacity: v, transform: `scale(${0.94 + 0.06 * v})` });
      const [tx, ty] = along(ease(ramp(s, 0.6, 3.8)));
      camera($('flowView'), $('flowStage'), 860, clamp(ty - 520, 80, 640), 1000);
      // The chart appears just ahead of the token, as the agent writes it.
      $('flowImg').style.webkitMaskImage = `linear-gradient(to bottom, #000 ${ty + 160}px, transparent ${ty + 420}px)`;
      style($('token'), {
        left: tx + 'px', top: ty + 'px',
        opacity: ramp(s, 0.5, 0.7) * (1 - ramp(s, 3.9, 4.2)),
        transform: `scale(${1 + Math.sin(s * 9) * 0.12})`,
      });
    }

    s = scene('agents', t);
    if (s >= 0) {
      headline('agents', s);
      const h = back(ramp(s, 0.2, 0.8));
      style($('hub'), { opacity: clamp(h), transform: `scale(${h}) rotate(${(1 - h) * -40}deg)` });
      let lines = '';
      document.querySelectorAll('.agent').forEach((el, i) => {
        const k = back(ramp(s, 0.5 + i * 0.15, 1.1 + i * 0.15));
        const angle = -Math.PI / 2 + (i * Math.PI) / 2 + s * 0.35;
        const r = 340 * k;
        const cx = 540 + Math.cos(angle) * r;
        const cy = 850 + Math.sin(angle) * r * 0.85;
        style(el, { left: cx + 'px', top: cy + 'px', opacity: clamp(k), transform: `translate(-50%, -50%) scale(${0.6 + 0.4 * k})` });
        const pulse = (s * 0.9 + i * 0.25) % 1;
        lines += `<line x1="540" y1="850" x2="${cx}" y2="${cy}" stroke="rgb(139 142 240 / ${0.4 * clamp(k)})" stroke-width="3" stroke-dasharray="6 10"/>`;
        lines += `<circle cx="${lerp(cx, 540, pulse)}" cy="${lerp(cy, 850, pulse)}" r="7" fill="#c9caff" opacity="${clamp(k) * Math.sin(pulse * Math.PI)}"/>`;
      });
      $('orbit').innerHTML = lines;
      const f = out(ramp(s, 1.4, 1.9));
      style($('agentsFoot'), { opacity: f, transform: `translateY(${(1 - f) * 30}px)` });
    }

    s = scene('open', t);
    if (s >= 0) {
      headline('open', s);
      [0, 1, 2].forEach((i) => {
        const k = back(ramp(s, 0.4 + i * 0.25, 1.0 + i * 0.25));
        style($('open' + i), { opacity: clamp(k), transform: `translateX(${(1 - k) * (i % 2 ? 300 : -300)}px) rotate(${(1 - k) * (i % 2 ? 3 : -3)}deg)` });
      });
      const type = (code, from) => code.slice(0, Math.floor(clamp(s - from, 0, 9) * 160));
      $('scenraCode').innerHTML = highlight(type(scenra, 0.9));
      $('embedCode').innerHTML = highlight(type(embed, 1.4)).replace(/scenraview/g, '<span class="k">scenraview</span>');
      $('stars').querySelectorAll('.tag').forEach((tag, i) => {
        const k = back(ramp(s, 1.0 + i * 0.12, 1.4 + i * 0.12));
        style(tag, { opacity: clamp(k), transform: `scale(${0.5 + 0.5 * k})` });
      });
    }

    s = scene('outro', t);
    if (s >= 0) {
      const m = back(ramp(s, 0, 0.7));
      style($('outroMark'), { opacity: clamp(m), transform: `scale(${m}) rotate(${(1 - m) * 30}deg) translateY(${Math.sin(s * 1.6) * 6}px)` });
      [['outroName', 0.25], ['outroLine', 0.45], ['outroZh', 0.6], ['outroCta', 0.85], ['outroFine', 1.05], ['outroUrl', 1.15]].forEach(([id, d]) => {
        const k = out(ramp(s, d, d + 0.5));
        const base = id === 'outroCta' ? 'translateX(-50%) ' : '';
        style($(id), { opacity: k, transform: `${base}translateY(${(1 - k) * 40}px)`, filter: `blur(${(1 - k) * 8}px)` });
      });
      const glow = 0.5 + 0.5 * Math.sin(s * 3);
      $('outroCta').style.boxShadow = `0 0 ${30 + glow * 40}px rgb(139 142 240 / ${0.4 + glow * 0.3})`;
    }
  };
}

// A small client for the Chrome DevTools Protocol.
async function devtools(port) {
  let url = '';
  for (let attempt = 0; attempt < 100 && !url; attempt++) {
    await new Promise((resolve) => setTimeout(resolve, 100));
    try {
      const list = await (await fetch(`http://127.0.0.1:${port}/json/list`)).json();
      url = list.find((target) => target.type === 'page')?.webSocketDebuggerUrl ?? '';
    } catch {}
  }
  if (!url) throw new Error('Chrome did not open a page.');
  const ws = new WebSocket(url);
  await new Promise((resolve, reject) => {
    ws.onopen = resolve;
    ws.onerror = reject;
  });
  let next = 0;
  const pending = new Map();
  ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    const call = pending.get(message.id);
    if (!call) return;
    pending.delete(message.id);
    if (message.error) call.reject(new Error(message.error.message));
    else call.resolve(message.result);
  };
  const send = (method, params = {}) =>
    new Promise((resolve, reject) => {
      const id = ++next;
      pending.set(id, { resolve, reject });
      ws.send(JSON.stringify({ id, method, params }));
    });
  return { send, close: () => ws.close() };
}

async function render(scratch, stills) {
  const file = path.join(scratch, 'video.html');
  writeFileSync(file, page());
  const port = 9300 + Math.floor(Math.random() * 600);
  const browser = spawn(
    chrome,
    [
      '--headless=new',
      `--remote-debugging-port=${port}`,
      `--user-data-dir=${path.join(scratch, 'chrome')}`,
      '--hide-scrollbars',
      '--force-device-scale-factor=1',
      '--allow-file-access-from-files',
      `--window-size=${W},${H}`,
      'about:blank',
    ],
    { stdio: 'ignore' },
  );
  const output = path.join(out, 'scenra.mp4');
  try {
    const cdp = await devtools(port);
    await cdp.send('Emulation.setDeviceMetricsOverride', { width: W, height: H, deviceScaleFactor: 1, mobile: false });
    await cdp.send('Page.navigate', { url: `file://${file}` });
    const ready = `(async () => {
      while (document.readyState !== 'complete' || !window.render) await new Promise((r) => setTimeout(r, 50));
      await Promise.all([...document.images].map((img) => img.decode()));
      await document.fonts.ready;
      return true;
    })()`;
    const loaded = await cdp.send('Runtime.evaluate', { expression: ready, awaitPromise: true });
    if (loaded.exceptionDetails) throw new Error(loaded.exceptionDetails.exception?.description ?? 'The page failed to load.');
    const frame = async (t, format) => {
      const result = await cdp.send('Runtime.evaluate', { expression: `render(${t})` });
      if (result.exceptionDetails) throw new Error(result.exceptionDetails.exception?.description);
      const { data } = await cdp.send('Page.captureScreenshot', format === 'png' ? { format } : { format, quality: 95 });
      return Buffer.from(data, 'base64');
    };

    if (stills) {
      for (const at of stills) writeFileSync(path.join(out, `still-${at}.png`), await frame(at, 'png'));
      cdp.close();
      return;
    }
    const audio = path.join(scratch, 'music.wav');
    const cuts = Object.values(times).map(([start]) => start).filter((start) => start > 0 && start < times.outro[0]);
    writeFileSync(audio, music({ duration: DURATION, cuts, outro: times.outro[0] }));
    const ffmpeg = spawn(
      'ffmpeg',
      ['-y', '-loglevel', 'error', '-f', 'image2pipe', '-framerate', String(FPS), '-i', '-', '-i', audio,
        '-c:a', 'aac', '-b:a', '192k', '-shortest', '-c:v', 'libx264', '-preset', 'slow', '-crf', '18', '-pix_fmt', 'yuv420p', '-movflags', '+faststart', output],
      { stdio: ['pipe', 'inherit', 'inherit'] },
    );
    const done = new Promise((resolve, reject) =>
      ffmpeg.on('exit', (code) => (code === 0 ? resolve() : reject(new Error(`ffmpeg exited with ${code}`)))),
    );
    for (let i = 0; i < DURATION * FPS; i++) {
      if (!ffmpeg.stdin.write(await frame(i / FPS, 'jpeg'))) {
        await new Promise((resolve) => ffmpeg.stdin.once('drain', resolve));
      }
      if (i % FPS === 0) process.stdout.write(`\r${i / FPS}s / ${DURATION}s`);
    }
    ffmpeg.stdin.end();
    await done;
    cdp.close();
    console.log(`\rwrote ${path.relative(website, output)}        `);
  } finally {
    browser.kill('SIGTERM');
  }
}

const stills = process.argv.find((arg) => arg.startsWith('--stills='))?.slice(9).split(',').map(Number);
const scratch = mkdtempSync(path.join(os.tmpdir(), 'scenra-video-'));
mkdirSync(out, { recursive: true });
try {
  await render(scratch, stills);
} finally {
  rmSync(scratch, { recursive: true, force: true });
}
