import { t as tr, setLang, getLang } from './i18n/index.js';
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
const demoAgents = () => [
  { id: 'orquestrador', name: tr('mock.orchestrator'), role: tr('mock.director'), emblem: 'gestao', face: 'av_p_robo', client: 'claude', tier: 'senior' },
  { id: 'arquiteto', name: tr('mock.architect'), role: tr('mock.planning'), emblem: 'planejamento', face: 'av_h2', client: 'claude', tier: 'medium', parent: 'orquestrador' },
  { id: 'engenheiro', name: tr('mock.go_engineer'), role: tr('mock.implementation'), emblem: 'codigo', face: 'av_h4', client: 'codex', tier: 'medium', parent: 'orquestrador' },
  { id: 'revisor', name: tr('mock.qa_reviewer'), role: tr('mock.testing_and_review'), emblem: 'revisao', face: 'av_f2', client: 'opencode', tier: 'medium', parent: 'orquestrador' },
  { id: 'portador', name: tr('mock.python_go_porter'), role: tr('mock.python_port'), emblem: 'producao', face: 'av_h7', client: 'codex', tier: 'senior', parent: 'engenheiro' },
];
let agents = demoAgents();
let generation = 0;
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
  emblems: ['gestao', 'planejamento', 'codigo', 'revisao', 'dados', 'pesquisa', 'visual', 'criacao', 'producao', 'ia'], tiers: ['junior', 'medium', 'senior'], clocks: models.clocks });
const push = (l) => {
  l.at = now();
  (consoles[l.agent] ||= []).push(l);
  if (['text', 'error', 'end', 'cmd'].includes(l.kind) && l.agent !== '__test__') activity.unshift(l);
  emit('line', l);
};
const slug = (s) => s.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');

function fakeRun(id, prompt) {
  const version = generation;
  running[id] = true;
  emit('snapshot', snapshot());
  const script = [
    { kind: 'cmd', text: `${id} … "${prompt.slice(0, 50)}"` },
    { kind: 'init', text: tr('mock.session_started_test_model') },
    { kind: 'tool', tool: 'Read', text: 'docs/ARQUITETURA.md' },
    { kind: 'text', text: tr('mock.understood_i_ll_handle', { p1: prompt }) },
    { kind: 'usage', text: tr('mock.tokens_1520_input_88_output') },
    { kind: 'end', text: tr('mock.session_ended_exit_0_0m03s_1608_tokens') },
  ];
  script.forEach((l, i) => setTimeout(() => {
    if (version !== generation) return;
    push({ agent: id, ...l });
    if (i === script.length - 1) { delete running[id]; emit('snapshot', snapshot()); }
  }, 450 * (i + 1)));
}

// ---- task board (mirrors internal/tasks closely enough to design the screens)
const git = { on: true, base: 'main', prefix: 'tao/', remote: true, gh: true, off: false, reason: '' };
const models = { auto: false, overrides: [], clocks: [{ client: 'opencode', model: 'opencode/big-pickle', until: new Date(Date.now() + 47 * 60000).toISOString(), reason: 'quota', source: 'informado' }] };
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
let flow = { on: false };
const boardView = () => ({
  flow,
  sprints: board.sprints.map((s) => ({ ...s, progress: progress(s.id) })),
  active: board.sprints.find((s) => s.status === 'active')?.id || '',
  backlog: progress(''),
  git,
  tasks: board.tasks.map((t) => ({ ...t, status_label: tr(`taskstatus.${t.status}`), assignee_name: agents.find((a) => a.id === t.assignee)?.name || '', running: !!taskRun[t.id],
    phases: ['start', 'work', 'submit', 'pr', 'review', 'verdict', 'merge'].map((p) => ({ phase: p, status: t.phases?.[p] || '' })) })),
});
const changed = () => emit('board', boardView());
const addTask = (n, by) => {
  const id = `T${String(board.tasks.length + 1).padStart(3, '0')}`;
  const t = { id, title: n.title, done: n.done || '', assignee: n.assignee || '', sprint: n.sprint || '', status: n.proposed ? 'proposed' : 'ready',
    priority: n.priority || 'normal', origin: n.origin || 'manual', created: ts(), phases: {} };
  hist(t, by, tr('mock.created'));
  board.tasks.push(t);
};
const work = (id, after) => {
  const version = generation;
  const t = findTask(id);
  taskRun[id] = true; t.status = 'in_progress'; t.phases.start = 'ok'; t.phases.work = 'running'; changed();
  fakeRun(t.assignee, `[${id}] ${t.title}`);
  setTimeout(() => { if (version !== generation) return; delete taskRun[id]; after(t); changed(); }, 3000);
};
function seed() {
  board.sprints.push({ id: 'S1', title: tr('mock.foundation'), goal: tr('mock.go_project_foundation'), status: 'closed', created: ts(), retro: ['ok'] },
    { id: 'S2', title: tr('mock.core_port'), goal: tr('mock.go_orchestrator_running_with_all_3_clients'), status: 'active', created: ts() });
  [[tr('mock.port_scheduler_py'), 'portador'], [tr('mock.port_agents_py'), 'portador'], [tr('mock.client_json_stream_reader'), 'engenheiro'],
    [tr('mock.python_go_parity_tests'), 'revisor']].forEach(([title, assignee]) => addTask({ title, assignee, sprint: 'S2', done: tr('mock.tests_pass') }, 'user'));
  board.tasks[0].status = 'in_progress'; board.tasks[0].question = tr('mock.may_i_start_porting_scheduler_py'); board.tasks[0].priority = 'high';
  board.tasks[2].status = 'review'; board.tasks[2].summary = tr('mock.stream_parse_covers_claude_codex_and_opencode');
}
seed();

const language = () => ({ current: getLang(), available: ['en', 'pt-BR'] });
const handlers = {
  Language: language,
  SetLanguage: (code) => {
    setLang(code);
    generation++;
    agents = demoAgents();
    for (const state of [consoles, running, taskRun]) for (const key of Object.keys(state)) delete state[key];
    activity.length = 0;
    board.tasks = [];
    board.sprints = [];
    seed();
    emit('language', language());
    emit('snapshot', snapshot());
    changed();
    return language();
  },
  Board: () => boardView(),
  StartFlow: (merge) => { flow = { on: true, merge, task: '', waiting: false }; changed(); },
  StopFlow: () => { flow = { on: false, reason: tr('mock.flow_finished') }; changed(); },
  SetTaskDeps: (id, deps) => { findTask(id).deps = deps; changed(); },
  NewSprint: (title, goal) => { board.sprints.push({ id: `S${board.sprints.length + 1}`, title, goal, status: 'planned', created: ts() }); changed(); },
  StartSprint: (id) => { if (board.sprints.some((s) => s.status === 'active')) throw new Error(tr('mock.a_sprint_is_already_active')); board.sprints.find((s) => s.id === id).status = 'active'; changed(); },
  CloseSprint: (id, retro, carry) => { const s = board.sprints.find((x) => x.id === id); s.status = 'closed'; s.retro = retro;
    board.tasks.filter((t) => t.sprint === id && t.status !== 'done').forEach((t) => { t.sprint = carry || ''; }); changed(); },
  AddTask: (n) => { addTask(n, 'user'); changed(); },
  UpdateTask: (t) => { Object.assign(findTask(t.id), t); hist(findTask(t.id), 'user', tr('mock.edited')); changed(); },
  ApproveTask: (id) => { findTask(id).status = 'ready'; changed(); },
  ArchiveTask: (id) => { findTask(id).archived = true; changed(); },
  ReopenTask: (id) => { Object.assign(findTask(id), { status: 'ready', verdicts: [] }); changed(); },
  SetTaskPriority: (id, high) => { findTask(id).priority = high ? 'high' : 'normal'; changed(); },
  RunTask: (id) => work(id, (t) => { t.status = 'review'; t.summary = tr('mock.done_simulated'); t.branch = `tao/${t.id.toLowerCase()}`; t.pr = 'https://github.com/exadmax/demo/pull/7';
    Object.assign(t.phases, { work: 'ok', submit: 'ok', pr: 'ok' }); hist(t, t.assignee, tr('mock.sent_for_review')); }),
  AnswerTask: (id, ans) => { const t = findTask(id); hist(t, 'user', tr('mock.reply', { p1: ans })); t.question = ''; handlers.RunTask(id); },
  DelegateTask: (id, to) => { const t = findTask(id); t.assignee = to; t.question = ''; hist(t, 'user', tr('mock.delegated_to', { p1: to })); handlers.RunTask(id); },
  ReviewTask: (id) => work(id, (t) => { t.approved = !!t.branch; t.status = t.branch ? 'review' : 'done'; t.phases.work = 'ok'; t.phases.review = 'ok'; t.phases.verdict = 'ok'; t.verdicts = ['revisor:approve']; hist(t, 'revisor', tr('mock.review_approved')); }),
  GenerateTasks: (agent, objective, sprint) => { const version = generation; fakeRun(agent, tr('mock.plan', { p1: objective }));
    setTimeout(() => { if (version !== generation) return; [tr('mock.map_the_module'), tr('mock.write_the_tests'), tr('mock.port_and_review')].forEach((title) => addTask({ title: `${title}: ${objective}`, sprint, assignee: 'engenheiro', origin: `agent:${agent}`, proposed: true }, agent)); changed(); }, 2500); },
  MergeTask: (id) => { const t = findTask(id); t.status = 'done'; t.phases.merge = 'ok'; hist(t, 'user', tr('mock.merged')); changed(); },
  GitStatus: () => git,
  SetGit: (g) => { Object.assign(git, g, { on: !g.off }); changed(); return git; },
  CanImport: () => ({ available: true, has_tasks: true }),
  ImportCards: () => ({ agents: 11, tasks: 10, sprints: 2, warnings: [] }),
  Models: () => ({ ...models, clients: catalog.map((c) => ({ id: c.id, name: c.name, models: c.models })) }),
  SetAutoSwitch: (on) => { models.auto = on; },
  SetModelOverride: (client, model, minutes, task) => { models.overrides.push({ key: task ? `${task}:${client}` : client, client, task, model, until: minutes ? new Date(Date.now() + minutes * 60000).toISOString() : null }); },
  ClearModelOverride: (key) => { models.overrides = models.overrides.filter((o) => o.key !== key); },
  ClearClock: (client, model) => { models.clocks = models.clocks.filter((c) => !(c.client === client && c.model === model)); },
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
  TestClient: (c, m, p) => fakeRun('__test__', p || tr('mock.hello')),
  StopAgent: (id) => { delete running[id]; },
  Tokens: () => agents.slice(0, 4).map((a, i) => ({ agent: a.id, name: a.name, client: tag(a.client).name, color: tag(a.client).color, model: view(a).model_in_use,
    status: 'ocioso', input: 180000 / (i + 1), output: 40000 / (i + 1), total: 220000 / (i + 1), sessions: 5 - i, share: [0.48, 0.24, 0.16, 0.12][i] })),
};

export const mock = {
  call: (name, ...args) => {
    if (!handlers[name]) return Promise.reject(new Error(tr('mock.mock', { p1: name })));
    try { return Promise.resolve(handlers[name](...args)); } catch (e) { return Promise.reject(e); }
  },
  on: (ev, fn) => { (listeners[ev] ||= []).push(fn); return () => { listeners[ev] = listeners[ev].filter((f) => f !== fn); }; },
};
