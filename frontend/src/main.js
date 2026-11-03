import { t as tr, setLang, getLang, LANGS } from './i18n/index.js';
import './style.css';
import { api, on, win } from './api.js';
import { h, icon, dialog, alertBox, toast, errText } from './ui.js';
import inicio from './views/inicio.js';
import mesa from './views/mesa.js';
import org from './views/org.js';
import sprints from './views/sprints.js';
import tokens from './views/tokens.js';
import novo from './views/novo.js';
import clientes from './views/clientes.js';
import consoleView from './views/console.js';
import { modelsWindow, gitWindow, importCards } from './views/models.js';

const VERSION = 'v0.4.0';
const tabs = () => [
  ['inicio', tr('tab.home'), inicio], ['mesa', tr('tab.agent_desk'), mesa], ['org', tr('tab.org_chart'), org], ['sprints', tr('tab.sprints'), sprints],
  ['tokens', tr('tab.tokens'), tokens], ['novo', tr('tab.new_agent'), novo], ['clientes', tr('tab.clients'), clientes], ['console', tr('tab.console'), consoleView],
];

// Shared app context handed to every view.
const ctx = {
  snap: { agents: [], clients: [], activity: [] },
  board: { sprints: [], tasks: [] },
  tab: 'inicio',
  agent: null, // the agent selected anywhere (org chart, console, desk)
  go(tab, agent) {
    if (agent !== undefined) ctx.agent = agent;
    ctx.tab = tab;
    renderPage();
  },
  async call(name, ...args) {
    try {
      const out = await api[name](...args);
      if (out && out.agents) setSnap(out);
      return out;
    } catch (e) {
      alertBox(errText(e), tr('common.error'));
      throw e;
    }
  },
  find: (id) => ctx.snap.agents.find((a) => a.id === id),
};

let page = null; // { el, update?(snap), line?(l) }
const root = document.getElementById('app');
let host, tabsEl, titleEl, status;
let closeMenus;

function setSnap(snap) {
  ctx.snap = snap;
  if (!ctx.agent || !ctx.find(ctx.agent)) ctx.agent = snap.agents?.[0]?.id || null;
  titleEl.textContent = tr('status.window_title', { title: tr('app.title'), project: snap.project || tr('status.no_project') });
  const installed = (snap.clients || []).filter((c) => c.installed).length;
  const active = (snap.agents || []).filter((a) => a.status === 'ativo').length;
  status.clients.replaceChildren(h('span.dot', { style: { background: installed ? 'var(--ok)' : 'var(--err)' } }), tr('status.connected_cli_clients', { p1: installed }));
  status.agents.replaceChildren(tr('status.agents_active', { p1: snap.agents?.length || 0, p2: active }));
  const clocks = snap.clocks || [];
  status.clocks.replaceChildren(clocks.length ? tr('status.model_s_out_of_quota', { p1: clocks.length }) : tr('status.quotas_ok'));
  drawTaskStatus();
  page?.update?.(snap);
}

function drawTaskStatus() {
  const open = (ctx.board?.tasks || []).filter((t) => t.status !== 'done' && !t.archived);
  const high = open.filter((t) => t.priority === 'high' || t.question).length;
  const errors = (ctx.snap.agents || []).filter((a) => a.status === 'erro').length;
  status.tasks.replaceChildren(tr('status.tasks_high_priority', { p1: open.length, p2: high }), errors ? tr('status.agent_s_with_errors', { p1: errors }) : '');
}

function setBoard(board) {
  ctx.board = board;
  drawTaskStatus();
  page?.board?.(board);
}

function renderPage() {
  tabsEl.replaceChildren(...tabs().map(([id, label]) => h(`div.tab${ctx.tab === id ? '.active' : ''}`, { onclick: () => ctx.go(id) }, label)));
  page?.destroy?.();
  const view = tabs().find(([id]) => id === ctx.tab)[2];
  page = view(ctx);
  host.replaceChildren(page.el);
}

// ---------------------------------------------------------------- menus
const menus = () => [
  [tr('menu.file'), [[tr('menu.open_project'), () => ctx.call('ChooseProject').then(() => api.Board().then(setBoard))],
    [tr('menu.import_cards_owners_json'), () => importCards(ctx)],
    [tr('menu.refresh'), () => { ctx.call('Snapshot'); api.Board().then(setBoard); }], null, [tr('menu.exit'), () => win.quit()]]],
  [tr('menu.agents'), [[tr('menu.new_agent_2'), () => ctx.go('novo')], [tr('menu.org_chart'), () => ctx.go('org')], [tr('menu.console'), () => ctx.go('console')], null,
    [tr('menu.find_installed_clients'), () => ctx.call('DetectClients').then(() => toast(tr('menu.clients_updated')))]]],
  [tr('menu.tasks'), [[tr('menu.sprints'), () => ctx.go('sprints')], [tr('menu.agent_desk_2'), () => ctx.go('mesa')]]],
  [tr('menu.tools'), [[tr('menu.models_and_quotas_2'), () => modelsWindow()], [tr('menu.git_workflow'), () => gitWindow(ctx)], [tr('menu.language'), LANGS.map(([code, label]) => [`${getLang() === code ? '✓ ' : '   '}${label}`, () => ctx.call('SetLanguage', code)])]]],
  [tr('menu.window'), [[tr('menu.home'), () => ctx.go('inicio')], [tr('menu.tokens'), () => ctx.go('tokens')], [tr('menu.clients'), () => ctx.go('clientes')]]],
  [tr('menu.help'), [[tr('menu.about_teamagent'), about]]],
];

function menubar() {
  const bar = h('div.menubar');
  let openMenu = null;
  const close = () => { openMenu?.classList.remove('open'); openMenu = null; };
  for (const [label, items] of menus()) {
    const menu = h('div.menu', h('u', label[0]), label.slice(1),
      h('div.dropdown.raised', items.map((it) => {
        if (!it) return h('hr');
        if (Array.isArray(it[1])) {
          const sub = h('div.dropdown.raised.language-menu', { style: { top: 0, left: '100%', display: 'none' } },
            it[1].map(([label, action]) => h('div.item', { onclick: (e) => { e.stopPropagation(); close(); action(); } }, label)));
          return h('div.item', { style: { position: 'relative' }, onclick: (e) => { e.stopPropagation(); sub.style.display = sub.style.display === 'none' ? 'block' : 'none'; },
            onmouseenter: () => { sub.style.display = 'block'; }, onmouseleave: () => { sub.style.display = 'none'; } }, it[0], ' ▸', sub);
        }
        return h('div.item', { onclick: (e) => { e.stopPropagation(); close(); it[1](); } }, it[0]);
      })));
    menu.addEventListener('click', (e) => { e.stopPropagation(); const was = openMenu === menu; close(); if (!was) { menu.classList.add('open'); openMenu = menu; } });
    menu.addEventListener('mouseenter', () => { if (openMenu && openMenu !== menu) { close(); menu.classList.add('open'); openMenu = menu; } });
    bar.append(menu);
  }
  closeMenus?.();
  document.addEventListener('click', close);
  closeMenus = () => document.removeEventListener('click', close);
  return bar;
}

function toolbar() {
  const tool = (name, title, fn) => h('div.tool', { title, onclick: fn }, icon(name, 24));
  return h('div.toolbar',
    tool('new', tr('toolbar.new_agent_3'), () => ctx.go('novo')), tool('open', tr('toolbar.open_project_2'), () => ctx.call('ChooseProject')),
    tool('save', tr('toolbar.everything_is_saved_automatically_in_tao_json'), () => toast(tr('toolbar.the_team_is_saved_automatically_in_tao'))), h('span.sep'),
    tool('team', tr('toolbar.org_chart'), () => ctx.go('org')), tool('robot', tr('toolbar.agent_desk_2'), () => ctx.go('mesa')),
    tool('tasks', tr('toolbar.sprints'), () => ctx.go('sprints')), h('span.sep'), tool('chart', tr('toolbar.tokens'), () => ctx.go('tokens')), h('span.sep'),
    tool('search', tr('toolbar.console'), () => ctx.go('console')), h('span.sep'), tool('help', tr('toolbar.about'), about));
}

function about() {
  dialog(tr('menu.about_teamagent_2'), (close) => [
    h('div.dialog-body.row', { style: { gap: '14px', alignItems: 'flex-start' } }, icon('robot', 48),
      h('div', h('b', { style: { font: 'bold 16px var(--pixel)' } }, tr('app.title')), h('br'), VERSION,
        h('p', tr('menu.orchestrates_agent_teams_claude_code_codex_cli'), h('br'), tr('menu.through_their_own_command_line_clients')),
        h('p.small.muted', tr('menu.go_port_of_jinchronicles_orq_github_com')))),
    h('div.dialog-btns', h('button.btn.default', { onclick: () => close() }, tr('common.ok'))),
  ]);
}

// ---------------------------------------------------------------- window
function buildWindow() {
  page?.destroy?.();
  page = null;
  host = h('div.page-host');
  tabsEl = h('div.tabs');
  titleEl = h('span.title');
  status = { clients: h('div'), agents: h('div'), tasks: h('div.grow'),
    clocks: h('div', { style: { cursor: 'pointer' }, title: tr('models.models_and_quotas'), onclick: () => modelsWindow() }),
    version: h('div', VERSION) };
  document.title = tr('app.title');
  root.replaceChildren(h('div.window.raised',
    h('div.titlebar', icon('app', 20), titleEl,
      h('div.tb-btns', h('button.btn.tb-btn', { onclick: () => win.minimise() }, '_'),
        h('button.btn.tb-btn', { onclick: () => win.toggleMaximise() }, '□'),
        h('button.btn.tb-btn', { onclick: () => win.quit() }, '✕'))),
    menubar(), toolbar(), tabsEl,
    h('div.raised', { style: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, margin: '0 2px' } }, host),
    h('div.statusbar', status.clients, status.agents, status.tasks, status.clocks, status.version)));
  setSnap(ctx.snap);
  renderPage();
}

async function start() {
  const language = await api.Language();
  setLang(language.current);
  buildWindow();
  on('language', (v) => { setLang(v.current); buildWindow(); });
  on('snapshot', setSnap);
  on('clients', (clients) => setSnap({ ...ctx.snap, clients }));
  on('line', (l) => page?.line?.(l));
  on('idle', (e) => page?.idle?.(e));
  on('board', setBoard);
  on('models', () => ctx.call('Snapshot'));

  await Promise.all([api.Snapshot(), api.Board()])
    .then(([s, b]) => { ctx.board = b; setSnap(s); renderPage(); })
    .catch((e) => { renderPage(); alertBox(errText(e)); });

}
start().catch((e) => { if (!host) buildWindow(); alertBox(errText(e)); });
