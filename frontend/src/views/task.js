// Task dialog: details, pipeline, history and the actions that move a task.
import { h, dialog, toast, hhmm } from '../ui.js';

const PHASE = { start: 'começar', work: 'trabalho', review: 'revisão', verdict: 'parecer' };
const SYM = { ok: '✓', failed: '✗', blocked: '?', running: '…', pending: '·' };

export function statusDot(t) {
  const st = t.question ? 'erro' : t.running ? 'ativo' : t.status === 'done' ? 'pausado' : t.status === 'in_progress' || t.status === 'review' ? 'ativo' : 'ocioso';
  return h(`span.dot.${st}`, { style: t.status === 'done' ? { background: '#555' } : {} });
}

export function askText(title, label, value = '') {
  return dialog(title, (close) => {
    const input = h('textarea.field', { rows: 4, style: { width: '440px' }, value });
    return [h('div.dialog-body.col', label, input),
      h('div.dialog-btns', h('button.btn.default', { onclick: () => close(input.value.trim()) }, 'OK'), h('button.btn', { onclick: () => close() }, 'Cancelar'))];
  });
}

export async function answer(ctx, t) {
  const text = await askText(`Responder ${t.id}`, h('div', h('b', `${t.assignee_name || 'O agente'} perguntou: `), t.question));
  if (text) { await ctx.call('AnswerTask', t.id, text); toast(`${t.id} voltou para ${t.assignee_name}.`); }
}

export async function delegate(ctx, t) {
  const r = await dialog(`Delegar ${t.id}`, (close) => {
    const sel = h('select.field', (ctx.snap.agents || []).filter((a) => a.id !== t.assignee && a.status !== 'pausado').map((a) => h('option', { value: a.id }, `${a.name} (${a.role})`)));
    const note = h('textarea.field', { rows: 3, style: { width: '420px' }, value: t.question || '' });
    return [h('div.dialog-body.col', 'Passar para:', sel, 'Observação (opcional):', note),
      h('div.dialog-btns', h('button.btn.default', { onclick: () => close({ to: sel.value, note: note.value.trim() }) }, 'Delegar'), h('button.btn', { onclick: () => close() }, 'Cancelar'))];
  });
  if (r?.to) { await ctx.call('DelegateTask', t.id, r.to, r.note); toast(`${t.id} delegada.`); }
}

export function openTask(ctx, id) {
  const t = (ctx.board?.tasks || []).find((x) => x.id === id);
  if (!t) return;
  return dialog(`${t.id} — ${t.title}`, (close) => {
    const act = (label, fn, opts = {}) => h(`button.btn${opts.def ? '.default' : ''}`, { disabled: opts.off, onclick: async () => { await fn(); close(); } }, label);
    const title = h('input.field', { value: t.title });
    const done = h('input.field', { value: t.done || '' });
    const who = h('select.field', h('option', { value: '' }, '—'), (ctx.snap.agents || []).map((a) => h('option', { value: a.id, selected: a.id === t.assignee }, a.name)));
    const rev = h('select.field', h('option', { value: '' }, 'automático (outro cliente)'), (ctx.snap.agents || []).map((a) => h('option', { value: a.id, selected: a.id === t.reviewer }, a.name)));
    const sprint = h('select.field', h('option', { value: '' }, '— sem sprint —'), (ctx.board?.sprints || []).filter((s) => s.status !== 'closed' || s.id === t.sprint)
      .map((s) => h('option', { value: s.id, selected: s.id === t.sprint }, `${s.id} — ${s.title}`)));
    const high = h('input', { type: 'checkbox', checked: t.priority === 'high' });
    const save = () => ctx.call('UpdateTask', { ...t, title: title.value.trim(), done: done.value.trim(), assignee: who.value, reviewer: rev.value, sprint: sprint.value, priority: high.checked ? 'high' : 'normal' });
    const busy = t.running;
    return [
      h('div.dialog-body.col', { style: { width: '640px' } },
        h('div.form', { style: { gridTemplateColumns: '110px 1fr' } },
          h('label', 'Atividade:'), title, h('label', 'Pronto quando:'), done, h('label', 'Responsável:'), who,
          h('label', 'Revisor:'), rev, h('label', 'Sprint:'), sprint, h('label', 'Prioridade:'), h('label.radio', high, 'alta'),
          h('label', 'Status:'), h('div.row', statusDot(t), h('b', busy ? 'Rodando…' : t.status_label), t.origin !== 'manual' ? h('span.muted', `· criada por ${t.origin.slice(6)}`) : '')),
        h('div.row', { style: { gap: '10px', flexWrap: 'wrap' } }, 'Etapas:', (t.phases || []).map((p) => h('span', { title: p.agent || '' }, `${SYM[p.status] || '·'} ${PHASE[p.phase]}${p.attempts > 1 ? ` (${p.attempts}×)` : ''}`))),
        t.question ? h('div.suggest.sunken', { style: { color: 'var(--err)' } }, h('b', 'Pergunta: '), t.question) : null,
        t.summary ? h('div', h('b', 'Último resultado: '), t.summary) : null,
        h('div', 'Histórico:'),
        h('div.list.sunken', { style: { maxHeight: '140px' } }, [...(t.history || [])].reverse().map((e) => h('div.li.small', h('b', hhmm(e.at)), `${e.by}: ${e.what}`)))),
      h('div.dialog-btns', { style: { flexWrap: 'wrap' } },
        t.status === 'proposed' ? act('Aprovar', () => ctx.call('ApproveTask', t.id), { def: true }) : null,
        t.question ? act('Responder…', () => answer(ctx, t), { def: true, off: busy }) : null,
        ['ready', 'in_progress'].includes(t.status) && !t.question ? act('Rodar agora', () => ctx.call('RunTask', t.id), { def: true, off: busy || !t.assignee }) : null,
        t.status === 'review' ? act('Revisar', () => ctx.call('ReviewTask', t.id), { def: true, off: busy }) : null,
        t.status !== 'done' ? act('Delegar…', () => delegate(ctx, t), { off: busy }) : null,
        t.status === 'done' ? act('Reabrir…', async () => { const r = await askText(`Reabrir ${t.id}`, 'Por que reabrir?'); if (r !== undefined) await ctx.call('ReopenTask', t.id, r); }) : null,
        act('Salvar', save),
        act('Arquivar', () => ctx.call('ArchiveTask', t.id), { off: busy }),
        h('button.btn', { onclick: () => close() }, 'Fechar')),
    ];
  }, { width: '680px' });
}
