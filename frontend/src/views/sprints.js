import { t as tr, codeLabel } from '../i18n/index.js';
// Sprints: sprint list on the left; the selected sprint's activities, manual creation and generation by an agent.
import { h, group, dialog, confirmBox, toast, originLabel } from '../ui.js';
import { openTask, statusDot } from './task.js';

export default function sprints(ctx) {
  const list = h('div.list.sunken', { style: { flex: 1 } });
  const head = h('div.panel-title');
  const goal = h('input.field.grow', { readOnly: true });
  const rows = h('tbody');
  const actions = h('div.row');
  const manualTitle = h('input.field.grow', { placeholder: tr('sprints.describe_the_task'), onkeydown: (e) => e.key === 'Enter' && addManual() });
  const manualWho = h('select.field');
  const genGoal = h('input.field.grow', { placeholder: tr('sprints.goal_e_g_port_the_logging_module'), onkeydown: (e) => e.key === 'Enter' && generate() });
  const genWho = h('select.field');
  const el = h('div.page',
    h('div.col', { style: { width: '220px', flex: 'none' } }, h('div', tr('sprints.project_sprints')), list,
      h('button.btn', { onclick: newSprint }, tr('sprints.new_sprint'))),
    h('div.col.grow', { style: { minWidth: 0 } },
      h('div.panel.raised.grow', head,
        h('div.panel-body.col', { style: { padding: '6px' } },
          h('div.row', h('b', tr('sprints.goal')), goal),
          h('div.sunken', { style: { flex: 1, overflow: 'auto', background: '#fff' } },
            h('table.table', h('thead', h('tr', [tr('sprints.id'), tr('sprints.task'), tr('sprints.assignee'), tr('sprints.status'), tr('sprints.origin'), ''].map((t) => h('th', t)))), rows)),
          actions)),
      h('div.row', { style: { alignItems: 'stretch' } },
        group(tr('sprints.new_task_manual'), h('div.col', manualTitle, h('div.row', manualWho, h('button.btn', { onclick: addManual }, tr('common.add'))))),
        group(tr('sprints.generate_with_agent'), h('div.col', genGoal, h('div.row', genWho, h('button.btn', { onclick: generate }, tr('sprints.generate'))),
          h('div.small.muted', tr('sprints.the_agent_breaks_the_goal_into_tasks')))))));

  let sel = null; // selected sprint id ("" = backlog)
  const B = () => ctx.board || { sprints: [], tasks: [] };

  function draw() {
    const b = B();
    if (sel === null || (sel && !b.sprints.some((s) => s.id === sel))) sel = b.active || b.sprints.at(-1)?.id || '';
    list.replaceChildren(
      ...b.sprints.map((s) => h(`div.li${s.id === sel ? '.sel' : ''}`, { style: { display: 'block' }, onclick: () => { sel = s.id; draw(); } },
        h('b', `${s.id} — ${s.title}`), s.status === 'active' ? ' ●' : '',
        h('div.small', tr('sprints.complete', { p1: s.progress.done, p2: s.progress.total, p3: ({ planned: tr('sprints.planned'), active: tr('sprints.active'), closed: tr('sprints.closed') })[s.status] })))),
      h(`div.li${sel === '' ? '.sel' : ''}`, { style: { display: 'block' }, onclick: () => { sel = ''; draw(); } },
        h('b', tr('sprints.no_sprint')), h('div.small', tr('sprints.tasks', { p1: b.backlog?.total || 0 }))));
    const sp = b.sprints.find((s) => s.id === sel);
    head.replaceChildren(sp ? tr('sprints.sprint', { p1: sp.id.slice(1), p2: sp.title }) : tr('sprints.tasks_without_a_sprint'),
      h('span.right', sp ? tr('sprints.complete_2', { p1: sp.progress.done, p2: sp.progress.total }) : ''));
    goal.value = sp?.goal || '';
    const tasks = b.tasks.filter((t) => (t.sprint || '') === sel && !t.archived);
    rows.replaceChildren(...(tasks.length ? tasks.map((t) => h('tr', { ondblclick: () => openTask(ctx, t.id), title: tr('sprints.double_click_to_open'), style: { cursor: 'default' } },
      h('td', h('b', t.id)), h('td', t.priority === 'high' ? h('b', { style: { color: 'var(--err)' } }, '! ') : '', t.title, t.question ? h('span.small', { style: { color: 'var(--err)' } }, tr('sprints.question')) : ''),
      h('td', t.assignee_name || '—'), h('td', h('div.row', statusDot(t), t.running ? tr('sprints.running') : codeLabel('taskstatus', t.status))),
      h('td', h('span', { style: { color: t.origin?.startsWith('agent:') ? 'var(--navy)' : '', fontWeight: t.origin?.startsWith('agent:') ? 'bold' : '' } }, originLabel(t.origin))),
      h('td', h('button.btn', { style: { minHeight: '18px', padding: '0 8px' }, onclick: () => openTask(ctx, t.id) }, tr('common.open')))))
      : [h('tr', h('td.empty', { colSpan: 6 }, tr('sprints.no_tasks_add_one_manually_or_generate')))]));
    actions.replaceChildren(...(sp ? [
      sp.status === 'planned' ? h('button.btn', { onclick: () => ctx.call('StartSprint', sp.id).then(() => toast(tr('sprints.started', { p1: sp.id }))) }, tr('sprints.start_sprint')) : null,
      sp.status !== 'closed' ? h('button.btn', { onclick: () => closeSprint(sp) }, tr('sprints.close_sprint')) : null,
      sp.retro?.length ? h('span.small.muted', tr('sprints.retro', { p1: sp.retro.join(' · ') })) : null].filter(Boolean) : []),
      ...flowControls(b.flow || {}));
    const opts = (s) => s.replaceChildren(...(ctx.snap.agents || []).map((a) => h('option', { value: a.id }, a.name)));
    const keepM = manualWho.value; const keepG = genWho.value;
    opts(manualWho); opts(genWho);
    manualWho.value = keepM || ctx.snap.agents?.[0]?.id || '';
    genWho.value = keepG || (ctx.snap.agents || []).find((a) => a.emblem === 'planejamento')?.id || ctx.snap.agents?.[0]?.id || '';
  }

  // The sequential flow: one card at a time, hand to hand, pausing only for the director.
  function flowControls(f) {
    const merge = h('input', { type: 'checkbox', checked: !!f.merge, disabled: !!f.on });
    const state = f.on ? (f.waiting ? f.reason : tr('flow.working', { p1: f.task || '—' })) : (f.reason || tr('flow.off'));
    return [h('span.right'),
      f.on ? null : h('label.small', merge, ' ', tr('flow.auto_merge')),
      h('button.btn', { onclick: () => (f.on ? ctx.call('StopFlow') : ctx.call('StartFlow', merge.checked)) }, f.on ? tr('flow.stop') : tr('flow.start')),
      h('span.small', { style: { color: f.waiting ? 'var(--err)' : '' } }, state)].filter(Boolean);
  }

  async function addManual() {
    if (!manualTitle.value.trim()) return;
    await ctx.call('AddTask', { title: manualTitle.value.trim(), assignee: manualWho.value, sprint: sel });
    manualTitle.value = '';
  }
  async function generate() {
    if (!genGoal.value.trim()) return;
    await ctx.call('GenerateTasks', genWho.value, genGoal.value.trim(), sel);
    toast(tr('sprints.is_planning_tasks_will_arrive_as_proposals', { p1: ctx.find(genWho.value)?.name }));
    genGoal.value = '';
  }
  async function newSprint() {
    const r = await dialog(tr('sprints.new_sprint'), (close) => {
      const title = h('input.field', { placeholder: tr('sprints.e_g_core_port') });
      const g = h('input.field', { placeholder: tr('sprints.what_will_be_ready_at_the_end') });
      return [h('div.dialog-body.form', { style: { width: '420px' } }, h('label', tr('sprints.title')), title, h('label', tr('sprints.goal')), g),
        h('div.dialog-btns', h('button.btn.default', { onclick: () => close({ title: title.value.trim(), goal: g.value.trim() }) }, tr('common.create')),
          h('button.btn', { onclick: () => close() }, tr('common.cancel')))];
    });
    if (r?.title) { await ctx.call('NewSprint', r.title, r.goal); sel = null; }
  }
  async function closeSprint(sp) {
    const open = B().tasks.filter((t) => t.sprint === sp.id && t.status !== 'done' && !t.archived).length;
    const r = await dialog(tr('sprints.close', { p1: sp.id }), (close) => {
      const retro = h('textarea.field', { rows: 4, style: { width: '420px' }, placeholder: tr('sprints.one_lesson_per_line') });
      const carry = h('select.field', h('option', { value: '' }, tr('sprints.no_sprint_2')),
        B().sprints.filter((s) => s.id !== sp.id && s.status !== 'closed').map((s) => h('option', { value: s.id }, `${s.id} — ${s.title}`)));
      return [h('div.dialog-body.col', tr('sprints.retrospective'), retro, open ? h('div', tr('sprints.move_open_task_s_to', { p1: open }), carry) : null),
        h('div.dialog-btns', h('button.btn.default', { onclick: () => close({ retro: retro.value.split('\n').map((x) => x.trim()).filter(Boolean), carry: carry.value }) }, tr('sprints.close_sprint_2')),
          h('button.btn', { onclick: () => close() }, tr('common.cancel')))];
    });
    if (r && (await confirmBox(tr('sprints.close_2', { p1: sp.id, p2: sp.title })))) await ctx.call('CloseSprint', sp.id, r.retro, r.carry);
  }

  draw();
  return { el, update: draw, board: draw };
}
