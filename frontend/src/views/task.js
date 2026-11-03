import { t as tr, codeLabel } from '../i18n/index.js';
// Task dialog: details, pipeline, history and the actions that move a task.
import { h, dialog, toast, hhmm, originDetail } from '../ui.js';

const SYM = { ok: '✓', failed: '✗', blocked: '?', running: '…', pending: '·', skipped: '–' };

export function statusDot(t) {
  const st = t.question ? 'erro' : t.running ? 'ativo' : t.status === 'done' ? 'pausado' : t.status === 'in_progress' || t.status === 'review' ? 'ativo' : 'ocioso';
  return h(`span.dot.${st}`, { title: codeLabel('taskstatus', t.status), style: t.status === 'done' ? { background: '#555' } : {} });
}

export function askText(title, label, value = '') {
  return dialog(title, (close) => {
    const input = h('textarea.field', { rows: 4, style: { width: '440px' }, value });
    return [h('div.dialog-body.col', label, input),
      h('div.dialog-btns', h('button.btn.default', { onclick: () => close(input.value.trim()) }, tr('common.ok')), h('button.btn', { onclick: () => close() }, tr('common.cancel')))];
  });
}

export async function answer(ctx, t) {
  const text = await askText(tr('task.reply_to', { p1: t.id }), h('div', h('b', tr('task.asked', { p1: t.assignee_name || tr('task.the_agent') })), t.question));
  if (text) { await ctx.call('AnswerTask', t.id, text); toast(tr('task.returned_to', { p1: t.id, p2: t.assignee_name })); }
}

export async function delegate(ctx, t) {
  const r = await dialog(tr('task.delegate', { p1: t.id }), (close) => {
    const sel = h('select.field', (ctx.snap.agents || []).filter((a) => a.id !== t.assignee && a.status !== 'pausado').map((a) => h('option', { value: a.id }, `${a.name} (${a.role})`)));
    const note = h('textarea.field', { rows: 3, style: { width: '420px' }, value: t.question || '' });
    return [h('div.dialog-body.col', tr('task.pass_to'), sel, tr('task.note_optional'), note),
      h('div.dialog-btns', h('button.btn.default', { onclick: () => close({ to: sel.value, note: note.value.trim() }) }, tr('task.delegate_2')), h('button.btn', { onclick: () => close() }, tr('common.cancel')))];
  });
  if (r?.to) { await ctx.call('DelegateTask', t.id, r.to, r.note); toast(tr('task.delegated', { p1: t.id })); }
}

export function openTask(ctx, id) {
  const t = (ctx.board?.tasks || []).find((x) => x.id === id);
  if (!t) return;
  return dialog(`${t.id} — ${t.title}`, (close) => {
    const act = (label, fn, opts = {}) => h(`button.btn${opts.def ? '.default' : ''}`, { disabled: opts.off, onclick: async () => { await fn(); close(); } }, label);
    const title = h('input.field', { value: t.title });
    const done = h('input.field', { value: t.done || '' });
    const who = h('select.field', h('option', { value: '' }, '—'), (ctx.snap.agents || []).map((a) => h('option', { value: a.id, selected: a.id === t.assignee }, a.name)));
    const rev = h('select.field', h('option', { value: '' }, tr('task.automatic_another_client')), (ctx.snap.agents || []).map((a) => h('option', { value: a.id, selected: a.id === t.reviewer }, a.name)));
    const sprint = h('select.field', h('option', { value: '' }, tr('task.no_sprint')), (ctx.board?.sprints || []).filter((s) => s.status !== 'closed' || s.id === t.sprint)
      .map((s) => h('option', { value: s.id, selected: s.id === t.sprint }, `${s.id} — ${s.title}`)));
    const high = h('input', { type: 'checkbox', checked: t.priority === 'high' });
    const save = () => ctx.call('UpdateTask', { ...t, title: title.value.trim(), done: done.value.trim(), assignee: who.value, reviewer: rev.value, sprint: sprint.value, priority: high.checked ? 'high' : 'normal' });
    const busy = t.running;
    return [
      h('div.dialog-body.col', { style: { width: '640px' } },
        h('div.form', { style: { gridTemplateColumns: '110px 1fr' } },
          h('label', tr('task.task')), title, h('label', tr('task.done_when')), done, h('label', tr('task.assignee')), who,
          h('label', tr('task.reviewer')), rev, h('label', tr('task.sprint')), sprint, h('label', tr('task.priority')), h('label.radio', high, tr('task.high')),
          h('label', tr('task.status')), h('div.row', statusDot(t), h('b', busy ? tr('task.running') : codeLabel('taskstatus', t.status)), h('span.muted', originDetail(t.origin)))),
        h('div.row', { style: { gap: '10px', flexWrap: 'wrap' } }, tr('task.phases'), (t.phases || []).map((p) => h('span', { title: [p.agent, p.model, p.detail].filter(Boolean).join(' · ') },
          `${SYM[p.status] || '·'} ${codeLabel('phase', p.phase)}${p.attempts > 1 ? ` (${p.attempts}×)` : ''}`))),
        t.branch ? h('div', h('b', tr('task.branch')), h('code', t.branch), t.pr ? [' · ', h('b', tr('task.pr_2')), t.pr] : null, t.approved ? h('b', { style: { color: 'var(--ok)' } }, tr('task.approved_ready_to_merge')) : null) : null,
        t.question ? h('div.suggest.sunken', { style: { color: 'var(--err)' } }, h('b', tr('task.question')), t.question) : null,
        t.summary ? h('div', h('b', tr('task.last_result')), t.summary) : null,
        h('div', tr('task.history')),
        h('div.list.sunken', { style: { maxHeight: '140px' } }, [...(t.history || [])].reverse().map((e) => h('div.li.small', h('b', hhmm(e.at)), tr('task.history_entry', { actor: ['user', 'diogo'].includes(e.by) ? tr('who.you') : e.by, what: e.what }))))),
      h('div.dialog-btns', { style: { flexWrap: 'wrap' } },
        t.status === 'proposed' ? act(tr('task.approve'), () => ctx.call('ApproveTask', t.id), { def: true }) : null,
        t.question ? act(tr('task.reply'), () => answer(ctx, t), { def: true, off: busy }) : null,
        ['ready', 'in_progress'].includes(t.status) && !t.question ? act(tr('task.run_now'), () => ctx.call('RunTask', t.id), { def: true, off: busy || !t.assignee }) : null,
        t.status === 'review' && !t.approved ? act(tr('task.review_2'), () => ctx.call('ReviewTask', t.id), { def: true, off: busy }) : null,
        t.status === 'review' && t.approved ? act(tr('task.merge_2'), async () => { await ctx.call('MergeTask', t.id); toast(tr('task.merged_into', { p1: t.id, p2: ctx.board?.git?.base || tr('git.base') })); }, { def: true, off: busy }) : null,
        t.status !== 'done' ? act(tr('task.delegate_3'), () => delegate(ctx, t), { off: busy }) : null,
        t.status === 'done' ? act(tr('task.reopen'), async () => { const r = await askText(tr('task.reopen_2', { p1: t.id }), tr('task.why_reopen')); if (r !== undefined) await ctx.call('ReopenTask', t.id, r); }) : null,
        act(tr('common.save'), save),
        act(tr('task.archive'), () => ctx.call('ArchiveTask', t.id), { off: busy }),
        h('button.btn', { onclick: () => close() }, tr('common.close'))),
    ];
  }, { width: '680px' });
}
