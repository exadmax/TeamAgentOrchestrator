import { t as tr, getLang } from '../i18n/index.js';
// Mesa do Agente: the agent at its desk saying what it needs, and the task board.
// Responder / Delegar / Arquivar act on the selected task; with no task they talk to the agent freely.
import { api } from '../api.js';
import { h, icon, dialog, sheetUrl, toast } from '../ui.js';
import { answer, delegate, openTask } from './task.js';

// Card kind of an open task, in the prototype's categories.
function kindOf(t) {
  if (t.priority === 'high') return { title: tr('desk.high_priority'), icon: 'alert', alert: true };
  if (t.question) return { title: tr('desk.support'), icon: 'headset' };
  if (t.status === 'review') return { title: tr('desk.follow_up'), icon: 'clip' };
  if (t.status === 'in_progress') return { title: tr('desk.analysis'), icon: 'bars' };
  return { title: tr('desk.new_task'), icon: 'star' };
}

// What the agent says and the suggested answer, from the task state.
function speech(t, a) {
  if (!t) return { say: a?.status === 'ativo' ? tr('desk.working_on_it') : tr('desk.hi_i_m_what_should_i_do', { p1: a?.name || tr('desk.the_agent') }),
    tip: tr('desk.choose_a_task_on_the_board_or') };
  if (t.running) return { say: tr('desk.working_on', { p1: t.id }), tip: tr('desk.follow_the_output_in_the_console_tab') };
  if (t.question) return { say: t.question, tip: tr('desk.reply_to_let_the_agent_continue_or'), err: /^(falhou|failed)/i.test(t.question) };
  switch (t.status) {
    case 'proposed': return { say: tr('desk.i_propose_may_i_proceed', { p1: t.title }), tip: tr('desk.approve_the_proposal_reply_or_archive_it') };
    case 'ready': return { say: tr('desk.may_i_start', { p1: t.id, p2: t.title }), tip: tr('desk.reply_tells_the_agent_to_start_now') };
    case 'in_progress': return { say: t.summary || tr('desk.i_m_working_on', { p1: t.title }), tip: tr('desk.reply_tells_the_agent_to_continue') };
    case 'review': if (t.approved) return { say: tr('desk.passed_review_may_i_merge_branch', { p1: t.id, p2: t.branch }), tip: tr('desk.reply_merges_the_task_through_the_pr') };
      return { say: tr('desk.finished', { p1: t.id, p2: t.summary || t.title }), tip: tr('desk.reply_requests_a_review_from_an_agent') };
    default: return { say: tr('desk.complete', { p1: t.id }), tip: tr('desk.all_done_archive_it_to_clear_the') };
  }
}

export default function mesa(ctx) {
  let taskId = null;
  const title = h('div.panel-title');
  const bubble = h('div.bubble');
  const suggest = h('div.suggest.sunken');
  const board = h('div.panel-body', { style: { background: 'var(--face)' } });
  const count = h('span.right', { style: { color: '#000' } });
  const pick = h('select.field', { onchange: (e) => { ctx.agent = e.target.value; taskId = null; draw(); } });
  const el = h('div.page',
    h('div.col.grow', { style: { minWidth: 0 } },
      h('div.panel.raised', title, h('div', { style: { padding: '6px' } },
        h('div.scene.sunken', { style: { backgroundImage: `url(${sheetUrl})`, backgroundSize: '157.4% auto', backgroundPosition: '13.26% 45%' } },
          // the wall sign is painted in Portuguese in the artwork: other languages get a translated sign over it
          getLang() === 'pt-BR' ? null : [h('div.scene-sign', h('b', tr('desk.sign.title')), h('span', tr('desk.sign.subtitle'))),
            h('div.scene-badge', tr('desk.sign.badge'))],
          bubble))),
      suggest,
      h('div.row', { style: { gap: '10px' } },
        h('button.btn.big.default.grow', { onclick: reply }, icon('reply', 24), tr('desk.reply')),
        h('button.btn.big.grow', { onclick: del }, icon('delegate', 24), tr('desk.delegate')),
        h('button.btn.big.grow', { onclick: archive }, icon('archive', 24), tr('desk.archive')))),
    h('div.panel.raised', { style: { width: '260px', flex: 'none' } },
      h('div.panel-title', { style: { background: 'none', color: '#000' } }, icon('cube', 24), tr('desk.task_board'), count), board));

  const task = () => (ctx.board?.tasks || []).find((t) => t.id === taskId);

  async function reply() {
    const t = task();
    const a = ctx.find(ctx.agent);
    if (!t) {
      const text = await dialog(tr('desk.reply'), (close) => {
        const input = h('textarea.field', { rows: 4, style: { width: '420px' } });
        return [h('div.dialog-body.col', tr('desk.instruction_for', { p1: a?.name }), input),
          h('div.dialog-btns', h('button.btn.default', { onclick: () => close(input.value.trim()) }, tr('common.send')), h('button.btn', { onclick: () => close() }, tr('common.cancel')))];
      });
      if (text) { await ctx.call('RunAgent', a.id, text); toast(tr('desk.received_the_instruction', { p1: a.name })); }
      return;
    }
    if (t.question) return answer(ctx, t);
    if (t.status === 'proposed') return ctx.call('ApproveTask', t.id).then(() => toast(tr('desk.approved', { p1: t.id })));
    if (t.status === 'review' && t.approved) return ctx.call('MergeTask', t.id).then(() => toast(tr('desk.merged_and_complete', { p1: t.id })));
    if (t.status === 'review') return ctx.call('ReviewTask', t.id).then(() => toast(tr('desk.sent_for_review', { p1: t.id })));
    if (t.status === 'ready' || t.status === 'in_progress') return ctx.call('RunTask', t.id).then(() => toast(tr('desk.started', { p1: t.assignee_name, p2: t.id })));
    return openTask(ctx, t.id);
  }
  async function del() {
    const t = task();
    if (t) return delegate(ctx, t);
    toast(tr('desk.choose_a_task_on_the_board_to'));
  }
  async function archive() {
    const t = task();
    if (!t) return;
    await ctx.call('ArchiveTask', t.id);
    taskId = null;
    toast(tr('desk.archived', { p1: t.id }));
  }

  function draw() {
    const open = (ctx.board?.tasks || []).filter((t) => t.status !== 'done' && !t.archived)
      .sort((x, y) => (y.priority === 'high') - (x.priority === 'high') || !!y.question - !!x.question || x.id.localeCompare(y.id));
    if (taskId && !open.some((t) => t.id === taskId)) taskId = null;
    const t = task();
    if (t?.assignee) ctx.agent = t.assignee;
    const a = ctx.find(ctx.agent);
    pick.replaceChildren(...(ctx.snap.agents || []).map((x) => h('option', { value: x.id, selected: x.id === ctx.agent }, x.name)));
    title.replaceChildren(icon('agent', 20), tr('desk.agent', { p1: a?.name || '—', p2: a?.client_name || '', p3: t ? ` · ${t.id}` : '' }), h('span.right', pick));
    const s = speech(t, a);
    bubble.textContent = s.say.length > 150 ? `${s.say.slice(0, 149)}…` : s.say;
    bubble.style.color = s.err ? 'var(--err)' : '';
    suggest.replaceChildren(icon('bulb', 24), h('div', h('b', tr('desk.suggested_reply')), s.tip));
    count.textContent = String(open.length);
    board.replaceChildren(...(open.length ? open.map((x) => {
      const k = kindOf(x);
      return h('div.taskcard.raised', { style: { outline: x.id === taskId ? '2px dotted #000' : '' }, onclick: () => { taskId = x.id; draw(); }, ondblclick: () => openTask(ctx, x.id) },
        h(`div.tc-title${k.alert ? '.alert' : ''}`, icon(k.icon, 18), h('span.grow', k.title),
          h('span', { title: tr('desk.archive'), style: { cursor: 'default' }, onclick: (e) => { e.stopPropagation(); ctx.call('ArchiveTask', x.id); } }, '✕')),
        h('div.tc-body', h('div', x.running ? '… ' : '', x.title), h('div.muted', `→ ${x.assignee_name || tr('desk.unassigned')}`)));
    }) : [h('div.empty.small', tr('desk.no_open_tasks_create_one_in_the'))]));
  }
  draw();
  return { el, update: draw, board: draw };
}
