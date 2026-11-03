import './style.css';
import { api, on } from './api.js';
import { h, icon, dialog, alertBox, toast, errText } from './ui.js';
import inicio from './views/inicio.js';
import mesa from './views/mesa.js';
import org from './views/org.js';
import sprints from './views/sprints.js';
import tokens from './views/tokens.js';
import novo from './views/novo.js';
import clientes from './views/clientes.js';
import consoleView from './views/console.js';

const VERSION = 'v0.1.0';
const TABS = [
  ['inicio', 'Início', inicio], ['mesa', 'Mesa do Agente', mesa], ['org', 'Organograma', org], ['sprints', 'Sprints', sprints],
  ['tokens', 'Tokens', tokens], ['novo', 'Novo Agente', novo], ['clientes', 'Clientes', clientes], ['console', 'Console', consoleView],
];

// Shared app context handed to every view.
const ctx = {
  snap: { agents: [], clients: [], activity: [] },
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
      alertBox(errText(e), 'Erro');
      throw e;
    }
  },
  find: (id) => ctx.snap.agents.find((a) => a.id === id),
};

let page = null; // { el, update?(snap), line?(l) }
const root = document.getElementById('app');
const host = h('div.page-host');
const tabsEl = h('div.tabs');
const status = { clients: h('div'), agents: h('div'), tasks: h('div.grow'), version: h('div', VERSION) };
const titleEl = h('span.title');

function setSnap(snap) {
  ctx.snap = snap;
  if (!ctx.agent || !ctx.find(ctx.agent)) ctx.agent = snap.agents?.[0]?.id || null;
  titleEl.textContent = `TeamAgent Orquestrador - ${snap.project || 'sem projeto'}`;
  const installed = (snap.clients || []).filter((c) => c.installed).length;
  const active = (snap.agents || []).filter((a) => a.status === 'ativo').length;
  status.clients.replaceChildren(h('span.dot', { style: { background: installed ? 'var(--ok)' : 'var(--err)' } }), `Conectado · ${installed} clientes CLI`);
  status.agents.replaceChildren(`Agentes: ${snap.agents?.length || 0} (${active} ativos)`);
  const errors = (snap.agents || []).filter((a) => a.status === 'erro').length;
  status.tasks.replaceChildren(errors ? `${errors} agente(s) com erro` : (snap.dir || ''));
  page?.update?.(snap);
}

function renderPage() {
  tabsEl.replaceChildren(...TABS.map(([id, label]) => h(`div.tab${ctx.tab === id ? '.active' : ''}`, { onclick: () => ctx.go(id) }, label)));
  page?.destroy?.();
  const view = TABS.find(([id]) => id === ctx.tab)[2];
  page = view(ctx);
  host.replaceChildren(page.el);
}

// ---------------------------------------------------------------- menus
const MENUS = [
  ['Arquivo', [['Abrir projeto…', () => ctx.call('ChooseProject')], ['Recarregar', () => ctx.call('Snapshot')], null, ['Sair', () => window.runtime?.Quit?.()]]],
  ['Agentes', [['Novo agente…', () => ctx.go('novo')], ['Organograma', () => ctx.go('org')], ['Console', () => ctx.go('console')], null,
    ['Procurar clientes instalados', () => ctx.call('DetectClients').then(() => toast('Clientes atualizados.'))]]],
  ['Tarefas', [['Sprints', () => ctx.go('sprints')], ['Mesa do agente', () => ctx.go('mesa')]]],
  ['Janela', [['Início', () => ctx.go('inicio')], ['Tokens', () => ctx.go('tokens')], ['Clientes', () => ctx.go('clientes')]]],
  ['Ajuda', [['Sobre o TeamAgent…', about]]],
];

function menubar() {
  const bar = h('div.menubar');
  let openMenu = null;
  const close = () => { openMenu?.classList.remove('open'); openMenu = null; };
  for (const [label, items] of MENUS) {
    const menu = h('div.menu', h('u', label[0]), label.slice(1),
      h('div.dropdown.raised', items.map((it) => (it ? h('div.item', { onclick: (e) => { e.stopPropagation(); close(); it[1](); } }, it[0]) : h('hr')))));
    menu.addEventListener('click', (e) => { e.stopPropagation(); const was = openMenu === menu; close(); if (!was) { menu.classList.add('open'); openMenu = menu; } });
    menu.addEventListener('mouseenter', () => { if (openMenu && openMenu !== menu) { close(); menu.classList.add('open'); openMenu = menu; } });
    bar.append(menu);
  }
  document.addEventListener('click', close);
  return bar;
}

function toolbar() {
  const tool = (name, title, fn) => h('div.tool', { title, onclick: fn }, icon(name, 24));
  return h('div.toolbar',
    tool('new', 'Novo agente', () => ctx.go('novo')), tool('open', 'Abrir projeto', () => ctx.call('ChooseProject')),
    tool('save', 'Tudo é salvo automaticamente em tao.json', () => toast('O time é salvo automaticamente em tao.json.')), h('span.sep'),
    tool('team', 'Organograma', () => ctx.go('org')), tool('robot', 'Mesa do agente', () => ctx.go('mesa')),
    tool('tasks', 'Sprints', () => ctx.go('sprints')), h('span.sep'), tool('chart', 'Tokens', () => ctx.go('tokens')), h('span.sep'),
    tool('search', 'Console', () => ctx.go('console')), h('span.sep'), tool('help', 'Sobre', about));
}

function about() {
  dialog('Sobre o TeamAgent', (close) => [
    h('div.dialog-body.row', { style: { gap: '14px', alignItems: 'flex-start' } }, icon('robot', 48),
      h('div', h('b', { style: { font: 'bold 16px var(--pixel)' } }, 'TeamAgent Orquestrador'), h('br'), VERSION,
        h('p', 'Orquestra times de agentes (Claude Code, Codex CLI, OpenCode, Gemini CLI)', h('br'), 'pelos próprios clientes de linha de comando.'),
        h('p.small.muted', 'Port em Go do orq do JinChronicles · github.com/exadmax/TeamAgentOrchestrator'))),
    h('div.dialog-btns', h('button.btn.default', { onclick: () => close() }, 'OK')),
  ]);
}

// ---------------------------------------------------------------- window
root.append(h('div.window.raised',
  h('div.titlebar', icon('app', 20), titleEl,
    h('div.tb-btns', h('button.btn.tb-btn', { onclick: () => window.runtime?.WindowMinimise?.() }, '_'),
      h('button.btn.tb-btn', { onclick: () => window.runtime?.WindowToggleMaximise?.() }, '□'),
      h('button.btn.tb-btn', { onclick: () => window.runtime?.Quit?.() }, '✕'))),
  menubar(), toolbar(), tabsEl,
  h('div.raised', { style: { flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, margin: '0 2px' } }, host),
  h('div.statusbar', status.clients, status.agents, status.tasks, status.version)));

on('snapshot', setSnap);
on('clients', (clients) => setSnap({ ...ctx.snap, clients }));
on('line', (l) => page?.line?.(l));
on('idle', (e) => page?.idle?.(e));

api.Snapshot().then((s) => { setSnap(s); renderPage(); }).catch((e) => { renderPage(); alertBox(errText(e)); });
