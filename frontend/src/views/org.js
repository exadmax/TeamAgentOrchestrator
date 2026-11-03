import { t as tr } from '../i18n/index.js';
// Organograma: the team tree on the left, the selected agent's properties on the right.
import { h, face, emblem, dot, clientTag, dialog, confirmBox, faceIds, toast, STATUS_LABEL, modelLabel } from '../ui.js';

export function facePicker(current) {
  let pick = current;
  return dialog(tr('org.customize_face'), (close) => {
    const grid = h('div.faces.sunken', faceIds.map((id) => {
      const opt = h(`div.opt${id === pick ? '.sel' : ''}`, { title: id, onclick: () => { grid.querySelectorAll('.opt').forEach((o) => o.classList.remove('sel')); opt.classList.add('sel'); pick = id; },
        ondblclick: () => close(id) }, face(id, 56));
      return opt;
    }));
    return [h('div.dialog-body', { style: { width: '520px' } }, grid),
      h('div.dialog-btns', h('button.btn.default', { onclick: () => close(pick) }, tr('common.ok')), h('button.btn', { onclick: () => close() }, tr('common.cancel')))];
  });
}

export default function org(ctx) {
  const chart = h('div.chart.sunken');
  const props = h('div.props');
  const el = h('div.page', chart, props);
  let draft = null; // edited copy of the selected agent

  function node(a) {
    const kids = ctx.snap.agents.filter((x) => (x.parent || '') === a.id);
    return h('li', h(`div.node.raised${a.id === ctx.agent ? '.sel' : ''}${a.parent ? '' : '.root'}`, { onclick: () => { ctx.agent = a.id; draw(); } },
      h('div.n-title', emblem(a.emblem, 16), a.name),
      h('div.n-body', h('div', a.role), h('div.n-model', modelLabel(a.model_in_use)),
        h('div.n-foot', dot(a.status), STATUS_LABEL[a.status] || a.status, h('span.grow'), clientTag(a.client_name, a.client_color)))),
    kids.length ? h('ul', kids.map(node)) : null);
  }

  function field(label, input) { return [h('label', label), input]; }
  function drawProps() {
    const a = ctx.find(ctx.agent);
    if (!a) { props.replaceChildren(h('div.raised.empty', tr('org.no_agents'))); return; }
    draft = { ...a };
    const set = (k) => (e) => { draft[k] = e.target.value; };
    const clients = ctx.snap.clients || [];
    const client = clients.find((c) => c.id === draft.client);
    const models = h('datalist#models', (client?.models || []).map((m) => h('option', { value: m })));
    const others = ctx.snap.agents.filter((x) => x.id !== a.id);
    props.replaceChildren(
      h('div.panel.raised', h('div.panel-title', tr('org.properties')), h('div.panel-body',
        h('div.form',
          field(tr('org.name'), h('input.field', { value: a.name, oninput: set('name') })),
          field(tr('org.role'), h('input.field', { value: a.role || '', oninput: set('role') })),
          field(tr('org.client'), h('select.field', { onchange: set('client') }, clients.map((c) => h('option', { value: c.id, selected: c.id === a.client }, `${c.name}${c.installed ? '' : tr('org.not_installed')}`)))),
          field(tr('org.tier'), h('select.field', { onchange: set('tier') }, ctx.snap.tiers.map((t) => h('option', { value: t, selected: t === a.tier }, t)))),
          field(tr('org.model'), h('input.field', { value: a.model || '', placeholder: a.model_in_use ? modelLabel(a.model_in_use) : tr('org.from_tier'), list: 'models', oninput: set('model') })), models,
          field(tr('org.manager'), h('select.field', { onchange: set('parent') }, h('option', { value: '' }, '—'), others.map((o) => h('option', { value: o.id, selected: o.id === a.parent }, o.name)))),
          field(tr('org.backup'), h('select.field', { title: tr('org.who_takes_over_when_this_agent_s'), onchange: set('fallback') },
            h('option', { value: '' }, tr('org.none')), others.map((o) => h('option', { value: o.id, selected: o.id === a.fallback }, `${o.name} (${o.client_name || o.client})`)))),
          field(tr('org.emblem'), h('div.row', emblem(a.emblem, 22), h('select.field.grow', { onchange: set('emblem') }, ctx.snap.emblems.map((e) => h('option', { value: e, selected: e === a.emblem }, e))))),
          field(tr('org.status'), h('div.row', dot(a.status), STATUS_LABEL[a.status]))),
        h('div', { style: { margin: '8px 0 3px' } }, tr('org.agent_instructions')),
        h('textarea.field', { rows: 3, style: { width: '100%' }, value: a.instructions || '', placeholder: tr('org.what_this_agent_should_always_know_or'), oninput: set('instructions') }),
        h('div', { style: { margin: '8px 0 3px' } }, tr('org.command')),
        h('div.cmd.sunken', `> ${a.command || ''}`),
        h('div.row', { style: { justifyContent: 'flex-end', marginTop: '8px' } }, h('button.btn.default', { onclick: save }, tr('org.apply'))))),
      h('div.col',
        h('button.btn', { onclick: async () => { const f = await facePicker(a.face); if (f) { draft.face = f; save(); } } }, face(a.face, 22), tr('org.customize_face_2')),
        h('button.btn', { onclick: () => ctx.go('console', a.id) }, tr('org.open_console')),
        h('button.btn', { onclick: () => ctx.call('TogglePause', a.id) }, a.paused ? tr('org.activate') : tr('org.pause')),
        h('button.btn', { onclick: () => { ctx.newParent = a.id; ctx.go('novo'); } }, tr('org.new_report')),
        h('button.btn', { onclick: remove }, tr('common.remove'))));
  }

  async function save() {
    const { status, model_in_use, client_name, client_color, last_line, command, ...agent } = draft;
    await ctx.call('UpdateAgent', agent);
    toast(tr('org.saved', { p1: agent.name }));
  }
  async function remove() {
    const a = ctx.find(ctx.agent);
    const subs = ctx.snap.agents.filter((x) => x.parent === a.id).length;
    if (!(await confirmBox(tr('org.remove', { p1: a.name, p2: subs ? tr('org.the_report_s_will_report_to_their', { p1: subs }) : '' })))) return;
    await ctx.call('RemoveAgent', a.id);
  }

  function draw() {
    const roots = (ctx.snap.agents || []).filter((a) => !a.parent);
    chart.replaceChildren(roots.length ? h('ul.tree', roots.map(node)) : h('div.empty', tr('org.no_agents_use_new_agent')));
    drawProps();
  }
  draw();
  // a wide team opens centered on the top of the chart (the tree is centered inside a wider scroll area)
  requestAnimationFrame(() => { chart.scrollLeft = (chart.scrollWidth - chart.clientWidth) / 2; });
  return { el, update: draw };
}
