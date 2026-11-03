// Início: the office floor (one desk per agent with its monitor showing the live output) and the side panels.
import { api } from '../api.js';
import { h, face, emblem, dot, panel, icon, hhmm } from '../ui.js';

const SCREEN_LINES = 5;

export default function inicio(ctx) {
  const lines = {}; // agent -> last lines
  const floor = h('div.office-floor');
  const side = h('div.side');
  const el = h('div.page', h('div.office.sunken', floor), side);

  const screenText = (l) => (l.kind === 'tool' ? `> ${l.tool} ${l.text || ''}` : l.kind === 'cmd' ? `C:\\> ${l.text}` : `> ${l.text}`);
  function screen(a) {
    const ls = (lines[a.id] || []).slice(-SCREEN_LINES);
    return h('div.screen', h('div.t', a.role || a.name),
      ls.length ? ls.map((l) => h('div', { style: { color: l.error ? 'var(--term-red)' : '' } }, screenText(l)))
        : h('div', a.status === 'pausado' ? '> pausado' : '> aguardando instrução…'), h('div.cursor', '_'));
  }
  function desk(a) {
    return h(`div.desk.${a.status}`, { title: `Abrir o console de ${a.name}`, onclick: () => ctx.go('console', a.id) },
      h('div.monitor', screen(a)), h('div.table-top'), h('div.avatar', face(a.face, 84)),
      h('div.plate', emblem(a.emblem, 22), dot(a.status), h('div', h('b', a.name), h('small', a.role))));
  }

  function draw() {
    const s = ctx.snap;
    floor.replaceChildren(
      h('div.office-head', h('div.logo', icon('robot', 24), h('div', h('b', 'TeamAgent'), h('small', s.project || ''))),
        h('div.hint', 'Clique num computador para falar com o agente')),
      ...(s.agents || []).map(desk));
    const count = (st) => (s.agents || []).filter((a) => a.status === st).length;
    const total = (s.agents || []).length || 1;
    const sp = (ctx.board?.sprints || []).find((x) => x.status === 'active');
    const busy = sp ? sp.progress.pct : Math.round((count('ativo') / total) * 100);
    side.replaceChildren(
      panel('Projeto Atual', [h('b', { style: { font: 'bold 14px var(--pixel)' } }, s.project || '—'), h('div.small.muted', { style: { wordBreak: 'break-all' } }, s.dir),
        h('div.row', { style: { marginTop: '8px' } }, h('div.progress.sunken.grow', Array.from({ length: Math.round(busy / 6.25) }, () => h('i'))), h('b', `${busy}%`)),
        h('div.small.muted', sp ? `${sp.id} ${sp.title}: ${sp.progress.done}/${sp.progress.total} atividades` : 'do time trabalhando agora (nenhuma sprint ativa)')]),
      panel('Resumo do Time', [['ativo', 'trabalhando'], ['ocioso', 'ociosos'], ['pausado', 'pausados'], ['erro', 'com erro']]
        .map(([st, label]) => h('div.row', { style: { margin: '3px 0' } }, h(`span.dot.${st}`, { style: { borderRadius: 0, width: '12px', height: '12px' } }), `${count(st)} ${label}`))),
      panel('Atividade Recente', h('div.list.sunken.activity', { style: { height: '100%' } },
        (s.activity || []).length ? s.activity.map((l) => h(`div.ev${l.error ? '.err' : ''}`, h('b', hhmm(l.at)),
          h('div', h('b', `${ctx.find(l.agent)?.name || l.agent}: `), h('span.txt', l.kind === 'cmd' ? 'recebeu uma instrução' : l.text))))
          : h('div.empty', 'Nada ainda. Mande uma instrução pelo Console.')), { style: { flex: 1, minHeight: 0 } }));
  }

  // Load what each agent printed so far, then keep it live.
  Promise.all((ctx.snap.agents || []).map((a) => api.Console(a.id).then((ls) => { lines[a.id] = ls || []; }))).then(draw);
  draw();
  return {
    el,
    update: draw,
    board: draw,
    line(l) {
      (lines[l.agent] ||= []).push(l);
      if (lines[l.agent].length > 50) lines[l.agent].shift();
      draw();
    },
  };
}
