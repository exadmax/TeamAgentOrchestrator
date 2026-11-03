// Browser-only stand-in for the Go App, used by `npm run dev` outside Wails.
const listeners = {};
const emit = (ev, data) => (listeners[ev] || []).forEach((fn) => fn(data));
const now = () => new Date().toISOString();

const catalog = [
  { id: 'claude', name: 'Claude Code', exe: 'claude', npm_package: '@anthropic-ai/claude-code', color: '#c15f3c', output: 'stream-json', models: ['haiku', 'sonnet', 'opus'], installed: true, version: '2.1.0', latest: '', path: 'C:\\npm\\claude.exe', model_flag: '--model', tiers: { junior: 'haiku', medium: 'sonnet', senior: 'opus' } },
  { id: 'codex', name: 'Codex CLI', exe: 'codex', npm_package: '@openai/codex', color: '#1a1a1a', output: 'json', models: ['gpt-6-luna', 'gpt-6.1-sol', 'gpt-6-astra'], installed: true, version: '0.166.0', latest: '', path: 'C:\\npm\\codex.exe', model_flag: '-m', tiers: { junior: 'gpt-6-luna', medium: 'gpt-6.1-sol', senior: 'gpt-6-astra' } },
  { id: 'opencode', name: 'OpenCode', exe: 'opencode', npm_package: '@opencode/cli', color: '#2e7d32', output: 'json', models: ['opencode/big-pickle'], installed: true, version: '2.2.0', latest: '', path: 'C:\\npm\\opencode.exe', model_flag: '-m', tiers: { junior: 'opencode/glm-5.3-flash-free', medium: 'opencode/big-pickle', senior: 'opencode/big-pickle' } },
  { id: 'gemini', name: 'Gemini CLI', exe: 'gemini', npm_package: '@google/gemini-cli', color: '#1a73e8', output: 'stream-json', models: ['gemini-3.8-flash'], installed: false, version: '', latest: '', path: '', model_flag: '-m', tiers: {} },
];
const tag = (c) => catalog.find((x) => x.id === c);
let agents = [
  { id: 'orquestrador', name: 'Orquestrador', role: 'Diretor do projeto', emblem: 'gestao', face: 'av_p_robo', client: 'claude', tier: 'senior' },
  { id: 'arquiteto', name: 'Arquiteto', role: 'Planejamento técnico', emblem: 'planejamento', face: 'av_h2', client: 'claude', tier: 'medium', parent: 'orquestrador' },
  { id: 'engenheiro', name: 'Engenheiro Go', role: 'Implementação', emblem: 'codigo', face: 'av_h4', client: 'codex', tier: 'medium', parent: 'orquestrador' },
  { id: 'revisor', name: 'Revisor QA', role: 'Testes e revisão', emblem: 'revisao', face: 'av_f2', client: 'opencode', tier: 'medium', parent: 'orquestrador' },
  { id: 'portador', name: 'Portador Py→Go', role: 'Port do Python', emblem: 'producao', face: 'av_h7', client: 'codex', tier: 'senior', parent: 'engenheiro' },
];
const consoles = {};
const running = {};
const activity = [];
const view = (a) => {
  const c = tag(a.client) || {};
  const model = a.model || c.tiers?.[a.tier] || '';
  const lines = consoles[a.id] || [];
  return { ...a, status: running[a.id] ? 'ativo' : a.paused ? 'pausado' : 'ocioso', model_in_use: model, client_name: c.name, client_color: c.color,
    last_line: lines.at(-1)?.text || '', command: `${c.exe} … ${c.model_flag} ${model} "<prompt>"` };
};
const snapshot = () => ({ project: 'TeamAgentOrchestrator', dir: 'D:\\workspace\\TeamAgentOrchestrator', agents: agents.map(view), clients: catalog, activity: activity.slice(0, 60),
  emblems: ['gestao', 'planejamento', 'codigo', 'revisao', 'dados', 'pesquisa', 'visual', 'criacao', 'producao', 'ia'], tiers: ['junior', 'medium', 'senior'], clocks: [] });
const push = (l) => {
  l.at = now();
  (consoles[l.agent] ||= []).push(l);
  if (['text', 'error', 'end', 'cmd'].includes(l.kind) && l.agent !== '__test__') activity.unshift(l);
  emit('line', l);
};
const slug = (s) => s.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');

function fakeRun(id, prompt) {
  running[id] = true;
  emit('snapshot', snapshot());
  const script = [
    { kind: 'cmd', text: `${id} … "${prompt.slice(0, 50)}"` },
    { kind: 'init', text: 'sessão iniciada · modelo de teste' },
    { kind: 'tool', tool: 'Read', text: 'docs/ARQUITETURA.md' },
    { kind: 'text', text: `Entendido. Vou cuidar de: ${prompt}` },
    { kind: 'usage', text: 'tokens: 1520 entrada · 88 saída' },
    { kind: 'end', text: 'sessão terminou (código 0) · 0m03s · 1608 tokens' },
  ];
  script.forEach((l, i) => setTimeout(() => {
    push({ agent: id, ...l });
    if (i === script.length - 1) { delete running[id]; emit('snapshot', snapshot()); }
  }, 450 * (i + 1)));
}

const handlers = {
  Snapshot: () => snapshot(),
  DetectClients: () => catalog,
  CheckUpdates: () => catalog.map((c) => ({ ...c, latest: c.version })),
  ChooseProject: () => snapshot(),
  Console: (id) => consoles[id] || [],
  AddAgent: (a) => { a.id ||= slug(a.name); if (agents.some((x) => x.id === a.id)) a.id += '-2'; agents.push(a); return snapshot(); },
  UpdateAgent: (a) => { agents = agents.map((x) => (x.id === a.id ? a : x)); return snapshot(); },
  RemoveAgent: (id) => { const p = agents.find((x) => x.id === id)?.parent; agents = agents.filter((x) => x.id !== id).map((x) => (x.parent === id ? { ...x, parent: p } : x)); return snapshot(); },
  TogglePause: (id) => { agents = agents.map((x) => (x.id === id ? { ...x, paused: !x.paused } : x)); return snapshot(); },
  RunAgent: (id, prompt) => fakeRun(id, prompt),
  TestClient: (c, m, p) => fakeRun('__test__', p || 'Olá'),
  StopAgent: (id) => { delete running[id]; },
  Tokens: () => agents.slice(0, 4).map((a, i) => ({ agent: a.id, name: a.name, client: tag(a.client).name, color: tag(a.client).color, model: view(a).model_in_use,
    status: 'ocioso', input: 180000 / (i + 1), output: 40000 / (i + 1), total: 220000 / (i + 1), sessions: 5 - i, share: [0.48, 0.24, 0.16, 0.12][i] })),
};

export const mock = {
  call: (name, ...args) => {
    if (!handlers[name]) return Promise.reject(new Error(`mock: ${name}`));
    try { return Promise.resolve(handlers[name](...args)); } catch (e) { return Promise.reject(e); }
  },
  on: (ev, fn) => { (listeners[ev] ||= []).push(fn); return () => { listeners[ev] = listeners[ev].filter((f) => f !== fn); }; },
};
