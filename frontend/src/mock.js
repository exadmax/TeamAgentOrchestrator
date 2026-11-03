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

// ---- task board (mirrors internal/tasks closely enough to design the screens)
const LABEL = { proposed: 'Proposta', ready: 'A fazer', in_progress: 'Fazendo', review: 'Em revisão', done: 'Concluída' };
const board = { sprints: [], tasks: [] };
const taskRun = {};
const ts = () => new Date().toISOString();
const findTask = (id) => board.tasks.find((t) => t.id === id);
const hist = (t, by, what) => { t.history = [...(t.history || []), { at: ts(), by, what }].slice(-50); t.updated = ts(); };
const progress = (sid) => {
  const ts_ = board.tasks.filter((t) => (t.sprint || '') === sid && !t.archived);
  const done = ts_.filter((t) => t.status === 'done').length;
  return { done, total: ts_.length, pct: ts_.length ? Math.round((done * 100) / ts_.length) : 0, blocked: ts_.filter((t) => t.question).map((t) => t.id) };
};
const boardView = () => ({
  sprints: board.sprints.map((s) => ({ ...s, progress: progress(s.id) })),
  active: board.sprints.find((s) => s.status === 'active')?.id || '',
  backlog: progress(''),
  tasks: board.tasks.map((t) => ({ ...t, status_label: LABEL[t.status], assignee_name: agents.find((a) => a.id === t.assignee)?.name || '', running: !!taskRun[t.id],
    phases: ['start', 'work', 'review', 'verdict'].map((p) => ({ phase: p, status: t.phases?.[p] || '' })) })),
});
const changed = () => emit('board', boardView());
const addTask = (n, by) => {
  const id = `T${String(board.tasks.length + 1).padStart(3, '0')}`;
  const t = { id, title: n.title, done: n.done || '', assignee: n.assignee || '', sprint: n.sprint || '', status: n.proposed ? 'proposed' : 'ready',
    priority: n.priority || 'normal', origin: n.origin || 'manual', created: ts(), phases: {} };
  hist(t, by, 'criada');
  board.tasks.push(t);
};
const work = (id, after) => {
  const t = findTask(id);
  taskRun[id] = true; t.status = 'in_progress'; t.phases.start = 'ok'; t.phases.work = 'running'; changed();
  fakeRun(t.assignee, `[${id}] ${t.title}`);
  setTimeout(() => { delete taskRun[id]; after(t); changed(); }, 3000);
};
(function seed() {
  board.sprints.push({ id: 'S1', title: 'Fundação', goal: 'Base do projeto em Go', status: 'closed', created: ts(), retro: ['ok'] },
    { id: 'S2', title: 'Porte do núcleo', goal: 'Orquestrador rodando em Go com os 3 clientes', status: 'active', created: ts() });
  [['Portar scheduler.py', 'portador'], ['Portar agents.py', 'portador'], ['Leitor de stream JSON dos clientes', 'engenheiro'],
    ['Testes de paridade Python × Go', 'revisor']].forEach(([title, assignee]) => addTask({ title, assignee, sprint: 'S2', done: 'testes passando' }, 'diogo'));
  board.tasks[0].status = 'in_progress'; board.tasks[0].question = 'Posso começar a portar o scheduler.py?'; board.tasks[0].priority = 'high';
  board.tasks[2].status = 'review'; board.tasks[2].summary = 'stream.Parse cobre claude, codex e opencode';
})();

const handlers = {
  Board: () => boardView(),
  NewSprint: (title, goal) => { board.sprints.push({ id: `S${board.sprints.length + 1}`, title, goal, status: 'planned', created: ts() }); changed(); },
  StartSprint: (id) => { if (board.sprints.some((s) => s.status === 'active')) throw new Error('já existe uma sprint ativa'); board.sprints.find((s) => s.id === id).status = 'active'; changed(); },
  CloseSprint: (id, retro, carry) => { const s = board.sprints.find((x) => x.id === id); s.status = 'closed'; s.retro = retro;
    board.tasks.filter((t) => t.sprint === id && t.status !== 'done').forEach((t) => { t.sprint = carry || ''; }); changed(); },
  AddTask: (n) => { addTask(n, 'diogo'); changed(); },
  UpdateTask: (t) => { Object.assign(findTask(t.id), t); hist(findTask(t.id), 'diogo', 'editada'); changed(); },
  ApproveTask: (id) => { findTask(id).status = 'ready'; changed(); },
  ArchiveTask: (id) => { findTask(id).archived = true; changed(); },
  ReopenTask: (id) => { Object.assign(findTask(id), { status: 'ready', verdicts: [] }); changed(); },
  SetTaskPriority: (id, high) => { findTask(id).priority = high ? 'high' : 'normal'; changed(); },
  RunTask: (id) => work(id, (t) => { t.status = 'review'; t.summary = 'feito (simulado)'; t.phases.work = 'ok'; hist(t, t.assignee, 'enviada para revisão'); }),
  AnswerTask: (id, ans) => { const t = findTask(id); hist(t, 'diogo', `resposta: ${ans}`); t.question = ''; handlers.RunTask(id); },
  DelegateTask: (id, to) => { const t = findTask(id); t.assignee = to; t.question = ''; hist(t, 'diogo', `delegada para ${to}`); handlers.RunTask(id); },
  ReviewTask: (id) => work(id, (t) => { t.status = 'done'; t.phases.review = 'ok'; t.phases.verdict = 'ok'; t.verdicts = ['revisor:approve']; hist(t, 'revisor', 'revisão: aprovada'); }),
  GenerateTasks: (agent, objective, sprint) => { fakeRun(agent, `planejar: ${objective}`);
    setTimeout(() => { ['Mapear o módulo', 'Escrever os testes', 'Portar e revisar'].forEach((title) => addTask({ title: `${title}: ${objective}`, sprint, assignee: 'engenheiro', origin: `agent:${agent}`, proposed: true }, agent)); changed(); }, 2500); },
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
