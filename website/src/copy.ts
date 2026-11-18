// The site's words, in English and Simplified Chinese. Short, specific and
// calm; every claim should be something the app does. Chinese headings,
// titles and kickers omit the trailing 。; body sentences keep it.

import type { Lang } from './site';
import type { Led, Tone } from './template';

type Feature = { led: Led; title: string; body: string };
type Note = { led: Led; title: string; body: string };
type Column = { value: string; title: string; body: string };
type Tab = { id: string; label: string; title: string; body: string; code: string; codeLang: 'text' };
type Section = { title: string; body: string[]; items: string[] };

// The Scenra files the format's tabs show, the same in both languages, as
// files keep their paths in English.
const files = {
  design: `design device=phone name="Phone/Now playing"
box class="flex-1 bg-zinc-950"
  use status-bar variant=dark class="px-6"
  icon chevron-down class="size-5 text-white/70" tap="close"
  image "Album art" src="assets/cover.jpg" class="size-64 rounded-3xl"
  text "Blue in Green" class="text-2xl font-bold text-white"
  box class="flex-row items-center justify-between px-8"
    icon skip-back class="size-7 text-white"
    box class="size-16 items-center justify-center rounded-full bg-white"
      icon pause class="size-7 text-zinc-900"
    end
    icon skip-forward class="size-7 text-white" note="Next in the playlist"
  end
end`,
  component: `component width=fill name="Phone/Mini player"
box name=player class="flex-row items-center gap-3 rounded-2xl bg-white p-2"
  image "Album art" name=cover class="size-10 rounded-xl"
  box class="flex-1"
    text "Blue in Green" name=title class="text-xs font-semibold"
    text "Miles Davis" name=artist class="text-[10px] text-zinc-400"
  end
  box name=toggle class="size-9 items-center justify-center rounded-full bg-rose-600"
    icon pause name=toggle-icon class="size-4 text-white"
  end
end
variant paused toggle-icon=play`,
  flow: `flow name="Phone: from Discover to the player"
node open "Open the app" kind=start
node feed "See the daily mix" note="The banner plays right away"
node pick "Tap where?" kind=decision
node list "Open a playlist"
node daily "Play the daily mix"
node play "Open the player"
open -> feed
feed -> pick
pick -> list "playlist card"
pick -> daily "daily mix"
list -> play "a song"
daily -> play`,
};

const en = {
  meta: {
    title: 'Scenra — design apps by talking',
    description:
      'Scenra is a desktop designer whose assistant, the coding agent you already use, draws an app’s screens and flow charts in plain-text files you keep. For macOS, Windows and Linux.',
    ogAlt: 'Scenra — design apps by talking',
  },

  nav: {
    skip: 'Skip to content',
    primary: 'Primary',
    home: 'Scenra home',
    links: [
      { label: 'Features', href: '#features' },
      { label: 'The assistant', href: '#assistant' },
      { label: 'Flows', href: '#flows' },
      { label: 'The format', href: '#format' },
    ],
    github: 'GitHub',
    cta: 'Download',
    menu: 'Open menu',
    close: 'Close menu',
    menuLabel: 'Menu',
  },

  theme: { label: 'Appearance', light: 'Light', dark: 'Dark', system: 'System' },
  language: { label: 'Language', en: 'English', zh: '中文' },
  code: { copy: 'Copy code', command: 'Copy command' },

  lightbox: {
    label: 'Screenshot',
    enlarge: 'Enlarge',
    close: 'Close',
    previous: 'Previous',
    next: 'Next',
  },

  hero: {
    badge: { tag: 'New', text: 'Scenra 0.1 for macOS, Windows and Linux' },
    title: 'Design apps.\nBy talking.',
    lede: 'Describe the app you have in mind. Scenra’s assistant, the coding agent you already use, draws its screens and flow charts as you agree, in plain-text files that live beside your code.',
    primary: 'Download Scenra',
    secondary: 'View on GitHub',
    fine: 'Free and open source · MIT · Updates itself',
    alt: 'Scenra: a music app’s desktop now playing screen on the canvas, the project’s designs and flows on the left, and the conversation that drew them on the right',
    caption: 'A music app’s now playing screen, and the conversation that drew it.',
    tone: 'violet' as Tone,
  },

  agents: {
    title: 'Drawn by the coding agent you already use',
    names: ['Claude Code', 'Codex', 'pi', 'CodeBuddy Code'],
    note: 'run with your own sign-ins, through Agio',
  },

  features: {
    title: 'Drafts that look like the app.',
    lead: 'Not wireframes in grey boxes: screens drawn from a few primitives shaped by Tailwind CSS classes, so a draft can look like anything.',
    items: [
      {
        led: 'violet',
        title: 'Phone, tablet and desktop',
        body: 'Boxes, texts, icons, images and SVG, laid out by Tailwind classes. Zoom, pan and select any element on the canvas.',
      },
      {
        led: 'green',
        title: 'Screens that play',
        body: 'Press an element to open another screen, close the one shown or switch tabs. A window’s content can be another design, so its sidebar stays as you play.',
      },
      {
        led: 'amber',
        title: 'Components, as in Figma',
        body: 'A tab bar is drawn once and used everywhere. A use picks variants, changes the texts and styles its component names, and fills its slots.',
      },
      {
        led: 'pink',
        title: 'Comments and notes',
        body: 'Rest the pointer on an element to read them: a comment asks the assistant for a change, a note records what the design cannot show.',
      },
      {
        led: 'blue',
        title: 'A tab for every folder',
        body: 'Each project opens in a tab of its own with its own conversations, beside a Home tab of the folders opened lately. All of them reopen at the next start.',
      },
      {
        led: 'teal',
        title: 'Your pictures, your Git',
        body: 'Pictures live in the folder’s assets/, and in a Git repository each file shows the lines it changed since the last commit.',
      },
    ] satisfies Feature[],
  },

  assistant: {
    title: 'Talk on the right. Draw in the middle.',
    lead: 'The assistant asks what it needs, then writes the folder’s files as the Scenra skill teaches it. Each file redraws as it writes.',
    alt: 'Scenra showing a phone app’s Discover screen beside the conversation, with the project’s phone and desktop designs listed on the left',
    caption: 'Phone designs share the folder with the desktop ones.',
    tone: 'pink' as Tone,
    items: [
      {
        led: 'violet',
        title: 'It sees what you see',
        body: 'Scenra tells the agent which file is open and what you selected, so “make this bigger” means the element you picked.',
      },
      {
        led: 'pink',
        title: 'Comments become changes',
        body: 'Leave comments on the elements you want changed, then ask. The assistant resolves them and keeps the notes true.',
      },
      {
        led: 'green',
        title: 'It reads your code',
        body: 'Conversations work in the open folder, so the agent can read the project and chart how the code there really works.',
      },
      {
        led: 'amber',
        title: 'One step to undo',
        body: 'Sending a message saves your changes, and each turn of the assistant undoes as one step, however many files it touched.',
      },
      {
        led: 'blue',
        title: 'Conversations that come back',
        body: 'A folder keeps its conversations, and opening it again resumes them. New ones start with the agent, model and reasoning level chosen last.',
      },
      {
        led: 'teal',
        title: 'Agents you choose',
        body: 'Settings shows the agents Scenra found and turns them or their models on and off. A small model of your choice can title new conversations.',
      },
    ] satisfies Feature[],
  },

  flows: {
    title: 'Flows of their own.',
    lead: 'What happens on a screen stays in its design. The workflows between screens, such as checkout or sign-up, are flow charts in flows/, all in one place.',
    alt: 'A Scenra flow chart from the music app’s home page to the player, with a decision to skip a song or collapse the player',
    caption: 'From home to the player: steps, a decision and a dashed way back.',
    tone: 'green' as Tone,
    notes: [
      { led: 'green', title: 'Steps', body: 'Starts, steps and ends, each with a note.' },
      { led: 'amber', title: 'Decisions', body: 'Branches labeled by the choice made.' },
      { led: 'violet', title: 'Layout', body: 'Laid out for you, top to bottom or left to right.' },
    ] satisfies Note[],
  },

  format: {
    title: 'Plain text you can read, diff and commit.',
    lead: 'No database and no lock-in. A .scenra file is a few readable lines, which an agent edits best and a pull request reviews well.',
    tabsLabel: 'Kinds of Scenra files',
    tabs: [
      {
        id: 'design',
        label: 'Design',
        title: 'mobile-now-playing.scenra',
        body: 'A screen at the top of the folder: elements shaped by classes, taps that play, and notes for what a drawing cannot show.',
        code: files.design,
        codeLang: 'text',
      },
      {
        id: 'component',
        label: 'Component',
        title: 'components/mini-player.scenra',
        body: 'A piece screens share. It names the elements a use may change and declares its variants, such as paused.',
        code: files.component,
        codeLang: 'text',
      },
      {
        id: 'flow',
        label: 'Flow',
        title: 'flows/mobile-feed-to-player.scenra',
        body: 'A workflow as nodes and edges. Scenra lays it out, so the file says only what happens.',
        code: files.flow,
        codeLang: 'text',
      },
    ] satisfies Tab[],
    link: 'Read the format',
  },

  folder: {
    title: 'One folder. One place for each kind.',
    lead: 'A project is flat, so a file’s name alone links to it, and moving it between folders breaks nothing.',
    items: [
      {
        value: 'designs',
        title: 'What the app opens',
        body: 'Windows and screens sit at the top of the folder, grouped in the sidebar by names such as Phone/Discover.',
      },
      {
        value: 'components/ · views/',
        title: 'What screens share',
        body: 'Components are drawn together under Components; views fill the outlets of their windows and play inside them.',
      },
      {
        value: 'flows/ · assets/',
        title: 'Workflows and pictures',
        body: 'Flow charts are listed atop the sidebar; pictures are named by their path, such as assets/latte.jpg.',
      },
    ] satisfies Column[],
  },

  library: {
    title: 'Put Scenra’s editor in your app.',
    lead: 'scenraview gives other NSGo apps the same editor and assistant: lay out its sidebar, canvas and conversation yourself, or take the whole window.',
    requirements: 'flowfile reads and writes the format without NSGo. The assistant runs the agents of an Agio backend.',
    readme: 'Read the README',
    examples: 'Browse the examples',
  },

  download: {
    title: 'Download Scenra.',
    lead: 'Installers for macOS on Apple silicon and for Windows and Linux on x64, on GitHub. Scenra checks for updates daily and installs them itself.',
    button: 'Download from GitHub',
    notes: 'Release notes',
    platforms: ['macOS', 'Windows', 'Linux'],
    terminal: 'Or install from a terminal',
    unix: 'macOS and Linux',
    windows: 'Windows',
    fine: 'Scenra runs the coding agents on your login shell’s PATH with your own sign-ins. Release builds send anonymous usage statistics, never designs, messages or paths, and the settings turn them off.',
    privacy: 'Read the privacy note',
  },

  built: {
    title: 'Built with NSGo and Agio',
    body: 'Scenra is native Go UI for desktops, written once with NSGo and shown in English and Chinese. Its assistant runs coding agents through Agio.',
    nsgo: 'nsgo.dev',
    agio: 'Agio',
  },

  footer: {
    note: 'Design apps by talking.',
    links: 'Project',
    readme: 'README',
    changelog: 'Changelog',
    issues: 'Issues',
    license: 'MIT License',
    privacy: 'Privacy',
  },

  privacy: {
    title: 'Privacy — Scenra',
    description: 'What Scenra and its website count, and what they never send.',
    heading: 'Privacy',
    lead: 'Scenra is built to know as little about you as possible. This page covers both the app and this website.',
    sections: [
      {
        title: 'The app',
        body: [
          'Release builds send anonymous usage statistics to a self-hosted Umami, so the project can tell how often and how long the app is used.',
        ],
        items: [
          'A pageview at start, an event for each message sent, and heartbeats while the designs or the conversation change.',
          'A random install ID kept on the device, so installs are counted without identifying people. The platform, language and window size go with each event.',
          'No design, message, file path, folder name or API key is ever sent.',
          'Turn statistics off under Settings → Statistics. Builds of go tool nsgo dev and test binaries send nothing.',
        ],
      },
      {
        title: 'This website',
        body: ['Pages are counted by the same Umami, with a cookieless script tagged website.'],
        items: [
          'It records page visits with the referring site, a coarse location derived from the request and the browser’s language.',
          'No cookies, fingerprints or advertising IDs are used, and nothing is shared with advertisers.',
          'The script runs only on the repository’s GitHub Pages address, never on local servers.',
        ],
      },
      {
        title: 'On your device',
        body: [],
        items: [
          'Settings, the open tabs, the folders opened lately, each folder’s conversations and the random statistics ID live in the app’s own storage.',
          'Designs live only in the folders you open; transcripts stay in the agents’ own sessions.',
        ],
      },
      {
        title: 'Agents and models',
        body: [
          'Your messages, and the files an agent reads, go to the provider of the agent you use, under that provider’s terms. The model you set to title conversations receives each conversation’s first message.',
        ],
        items: [],
      },
      {
        title: 'Contact',
        body: ['Questions or requests? Open an issue on GitHub.'],
        items: [],
      },
    ] satisfies Section[],
    back: 'Back to Scenra',
  },

  notFound: {
    title: 'Page not found — Scenra',
    heading: 'This screen isn’t in the flow.',
    body: 'The address is gone, or it never existed. Scenra lives on the home page.',
    back: 'Back to Scenra',
  },
};

const zh: typeof en = {
  meta: {
    title: 'Scenra — 把应用设计，说出来',
    description:
      'Scenra 是一款桌面设计工具：它的助手就是你已经在用的编程智能体，把应用的界面和流程图画进你自己保存的纯文本文件。支持 macOS、Windows 和 Linux。',
    ogAlt: 'Scenra — 把应用设计，说出来',
  },

  nav: {
    skip: '跳到内容',
    primary: '主导航',
    home: 'Scenra 首页',
    links: [
      { label: '功能', href: '#features' },
      { label: '助手', href: '#assistant' },
      { label: '流程', href: '#flows' },
      { label: '文件格式', href: '#format' },
    ],
    github: 'GitHub',
    cta: '下载',
    menu: '打开菜单',
    close: '关闭菜单',
    menuLabel: '菜单',
  },

  theme: { label: '外观', light: '浅色', dark: '深色', system: '跟随系统' },
  language: { label: '语言', en: 'English', zh: '中文' },
  code: { copy: '复制代码', command: '复制命令' },

  lightbox: {
    label: '截图',
    enlarge: '放大',
    close: '关闭',
    previous: '上一张',
    next: '下一张',
  },

  hero: {
    badge: { tag: '新', text: 'Scenra 0.1 支持 macOS、Windows 和 Linux' },
    title: '把应用设计\n说出来',
    lede: '说出你脑海里的应用。Scenra 的助手就是你已经在用的编程智能体，边商量边画出界面和流程图，存成放在代码旁边的纯文本文件。',
    primary: '下载 Scenra',
    secondary: '在 GitHub 查看',
    fine: '免费开源 · MIT · 自动更新',
    alt: 'Scenra：画布上是一款音乐应用的桌面端正在播放页，左边是项目的设计和流程，右边是画出它们的对话',
    caption: '音乐应用的正在播放页，以及画出它的那段对话。',
    tone: 'violet',
  },

  agents: {
    title: '由你已经在用的编程智能体来画',
    names: ['Claude Code', 'Codex', 'pi', 'CodeBuddy Code'],
    note: '通过 Agio，用你自己的登录状态运行',
  },

  features: {
    title: '草稿就像应用本身',
    lead: '不是灰色方框的线框图：界面由几种基本元素搭成，外观由 Tailwind CSS 类控制，草稿可以是任何样子。',
    items: [
      {
        led: 'violet',
        title: '手机、平板和桌面',
        body: '盒子、文字、图标、图片和 SVG，用 Tailwind 类排布。在画布上缩放、平移，选中任意元素。',
      },
      {
        led: 'green',
        title: '界面可以演示',
        body: '按下一个元素，就能打开另一个界面、关闭当前界面或切换标签。窗口的内容区可以是另一个设计，演示时侧边栏保持不动。',
      },
      {
        led: 'amber',
        title: '像 Figma 一样的组件',
        body: '标签栏画一次，处处使用。每次使用可以挑选变体，修改组件开放的文字和样式，并填充它留出的插槽。',
      },
      {
        led: 'pink',
        title: '评论与备注',
        body: '指针停在元素上就能看到：评论请助手做修改，备注记下设计画不出来的东西。',
      },
      {
        led: 'blue',
        title: '一个文件夹一个标签页',
        body: '每个项目在自己的标签页中打开，各有各的对话，旁边是列出最近文件夹的主页。下次启动时全部恢复。',
      },
      {
        led: 'teal',
        title: '你的图片，你的 Git',
        body: '图片放在文件夹的 assets/ 中；在 Git 仓库里，每个文件都会显示自上次提交以来增删的行数。',
      },
    ],
  },

  assistant: {
    title: '右边说，中间画',
    lead: '助手先问清需要什么，再按 Scenra 技能教的方式写文件夹里的文件。每个文件边写边重画。',
    alt: 'Scenra 显示手机应用的发现页和旁边的对话，左边列出项目的手机和桌面设计',
    caption: '手机设计与桌面设计放在同一个文件夹里。',
    tone: 'pink',
    items: [
      {
        led: 'violet',
        title: '你看到的，它也知道',
        body: 'Scenra 告诉智能体打开的是哪个文件、你选中了什么，所以“把这个放大”说的就是你选中的元素。',
      },
      {
        led: 'pink',
        title: '评论变成修改',
        body: '在想改的元素上留下评论，然后开口。助手逐条处理，并让备注保持准确。',
      },
      {
        led: 'green',
        title: '它会读你的代码',
        body: '对话在打开的文件夹中进行，智能体可以读取项目，把代码真实的运作方式画成流程。',
      },
      {
        led: 'amber',
        title: '一步撤销',
        body: '发送消息时保存你的修改；助手的每一轮无论改了多少文件，都作为一步撤销。',
      },
      {
        led: 'blue',
        title: '对话不会丢',
        body: '每个文件夹保存自己的对话，再次打开时自动恢复。新对话沿用上次选择的智能体、模型和推理档位。',
      },
      {
        led: 'teal',
        title: '智能体由你挑',
        body: '设置中列出 Scenra 找到的智能体，可以开关它们和它们的模型；还可以用一个你选的小模型为新对话起标题。',
      },
    ],
  },

  flows: {
    title: '流程自成一张图',
    lead: '界面上发生的事留在它的设计里；界面之间的工作流，例如结账或注册，是 flows/ 中的流程图，集中在一处。',
    alt: 'Scenra 流程图：音乐应用从首页到播放页，并在播放页选择切歌或收起',
    caption: '从首页到播放页：步骤、一个决策，和一条虚线的回路。',
    tone: 'green',
    notes: [
      { led: 'green', title: '步骤', body: '开始、步骤和结束，每个都可以带备注。' },
      { led: 'amber', title: '决策', body: '分支以所做的选择标注。' },
      { led: 'violet', title: '布局', body: '自动排布，自上而下或从左到右。' },
    ],
  },

  format: {
    title: '能读、能 diff、能提交的纯文本',
    lead: '没有数据库，也没有厂商锁定。.scenra 文件就是几行可读的文字，智能体改得好，pull request 也审得清。',
    tabsLabel: 'Scenra 文件的种类',
    tabs: [
      {
        id: 'design',
        label: '设计',
        title: 'mobile-now-playing.scenra',
        body: '放在文件夹顶层的界面：由类控制外观的元素、可以演示的点击，以及记下画不出来的东西的备注。',
        code: files.design,
        codeLang: 'text',
      },
      {
        id: 'component',
        label: '组件',
        title: 'components/mini-player.scenra',
        body: '界面之间共享的部件。它给可修改的元素命名，并声明自己的变体，例如 paused。',
        code: files.component,
        codeLang: 'text',
      },
      {
        id: 'flow',
        label: '流程',
        title: 'flows/mobile-feed-to-player.scenra',
        body: '由节点和连线组成的工作流。布局交给 Scenra，文件里只写发生了什么。',
        code: files.flow,
        codeLang: 'text',
      },
    ],
    link: '阅读格式说明',
  },

  folder: {
    title: '一个文件夹，每种文件各有其位',
    lead: '项目是扁平的，只凭文件名就能引用它，在文件夹之间移动也不会断开任何链接。',
    items: [
      {
        value: 'designs',
        title: '应用打开的界面',
        body: '窗口和界面放在文件夹顶层，在侧边栏里按 Phone/Discover 这样的名字分组。',
      },
      {
        value: 'components/ · views/',
        title: '界面共享的部分',
        body: '组件在“组件”中集中展示；子视图填充窗口的出口，在窗口里演示。',
      },
      {
        value: 'flows/ · assets/',
        title: '工作流与图片',
        body: '流程图列在侧边栏最上方；图片按路径引用，例如 assets/latte.jpg。',
      },
    ],
  },

  library: {
    title: '把 Scenra 的编辑器放进你的应用',
    lead: 'scenraview 为其他 NSGo 应用提供同样的编辑器和助手：自己排布侧边栏、画布和对话，或者直接用整个窗口。',
    requirements: 'flowfile 不依赖 NSGo 就能读写这种格式。助手运行 Agio 后端的智能体。',
    readme: '阅读 README',
    examples: '浏览示例',
  },

  download: {
    title: '下载 Scenra',
    lead: '在 GitHub 下载安装包：macOS 支持 Apple 芯片，Windows 和 Linux 支持 x64。Scenra 每天检查更新并自行安装。',
    button: '从 GitHub 下载',
    notes: '更新日志',
    platforms: ['macOS', 'Windows', 'Linux'],
    terminal: '或在终端中安装',
    unix: 'macOS 和 Linux',
    windows: 'Windows',
    fine: 'Scenra 用你自己的登录状态运行登录 shell 的 PATH 中的编程智能体。正式版本会发送匿名使用统计，从不包含设计、消息或路径，可在设置中关闭。',
    privacy: '阅读隐私说明',
  },

  built: {
    title: '用 NSGo 和 Agio 构建',
    body: 'Scenra 是用 NSGo 写一次的原生 Go 桌面界面，支持中文和英文。它的助手通过 Agio 运行编程智能体。',
    nsgo: 'nsgo.dev',
    agio: 'Agio',
  },

  footer: {
    note: '把应用设计，说出来。',
    links: '项目',
    readme: 'README',
    changelog: '更新日志',
    issues: '问题反馈',
    license: 'MIT 许可证',
    privacy: '隐私',
  },

  privacy: {
    title: '隐私 — Scenra',
    description: 'Scenra 及其网站统计什么，以及从不发送什么。',
    heading: '隐私',
    lead: 'Scenra 尽量少知道关于你的事。本页同时说明应用和网站的做法。',
    sections: [
      {
        title: '应用',
        body: ['正式版本会向自建的 Umami 发送匿名使用统计，以便了解应用被使用的频率和时长。'],
        items: [
          '启动时一次页面浏览，每发送一条消息一个事件，设计或对话变化期间发送心跳。',
          '设备上保存一个随机安装 ID，用于统计安装数量而不识别个人。每个事件会带上平台、语言和窗口尺寸。',
          '从不发送任何设计、消息、文件路径、文件夹名称或 API 密钥。',
          '可在设置 → 统计中关闭。go tool nsgo dev 的构建和测试程序不发送任何内容。',
        ],
      },
      {
        title: '本网站',
        body: ['页面由同一个 Umami 统计，使用无 Cookie 的脚本，并标记为 website。'],
        items: [
          '统计页面访问，附带来源站点、由请求得出的粗略位置和浏览器语言。',
          '不使用 Cookie、浏览器指纹或广告 ID，也不与广告商共享任何内容。',
          '脚本只在本仓库的 GitHub Pages 地址运行，不会在本地服务器上运行。',
        ],
      },
      {
        title: '你的设备上',
        body: [],
        items: [
          '设置、打开的标签页、最近打开的文件夹、各文件夹的对话和随机统计 ID 保存在应用自己的存储中。',
          '设计只保存在你打开的文件夹里；对话记录留在智能体自己的会话中。',
        ],
      },
      {
        title: '智能体与模型',
        body: [
          '你的消息以及智能体读取的文件，会发送给你所用智能体的服务商，并按该服务商的条款处理。你设置的对话标题模型会收到每段对话的第一条消息。',
        ],
        items: [],
      },
      {
        title: '联系',
        body: ['有疑问或请求？请在 GitHub 上提 issue。'],
        items: [],
      },
    ],
    back: '回到 Scenra',
  },

  notFound: {
    title: '页面未找到 — Scenra',
    heading: '这个界面不在流程里',
    body: '地址不在了，或者从未存在。Scenra 在首页。',
    back: '回到 Scenra',
  },
};

export const copies: Record<Lang, typeof en> = { en, zh };

export type Copy = typeof en;
