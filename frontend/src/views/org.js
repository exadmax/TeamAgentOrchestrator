// Organograma: the team tree on the left, the selected agent's properties on the right.
import { h, face, emblem, dot, clientTag, dialog, confirmBox, faceIds, toast, STATUS_LABEL } from '../ui.js';

export function facePicker(current) {
  let pick = current;
  return dialog('Personalizar rostinho', (close) => {
    const grid = h('div.faces.sunken', faceIds.map((id) => {
      const opt = h(`div.opt${id === pick ? '.sel' : ''}`, { title: id, onclick: () => { grid.querySelectorAll('.opt').forEach((o) => o.classList.remove('sel')); opt.classList.add('sel'); pick = id; },
        ondblclick: () => close(id) }, face(id, 56));
      return opt;
    }));
    return [h('div.dialog-body', { style: { width: '520px' } }, grid),
      h('div.dialog-btns', h('button.btn.default', { onclick: () => close(pick) }, 'OK'), h('button.btn', { onclick: () => close() }, 'Cancelar'))];
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
      h('div.n-body', h('div', a.role), h('div.n-model', a.model_in_use || 'modelo padrão'),
        h('div.n-foot', dot(a.status), STATUS_LABEL[a.status] || a.status, h('span.grow'), clientTag(a.client_name, a.client_color)))),
    kids.length ? h('ul', kids.map(node)) : null);
  }

  function field(label, input) { return [h('label', label), input]; }
  function drawProps() {
    const a = ctx.find(ctx.agent);
    if (!a) { props.replaceChildren(h('div.raised.empty', 'Nenhum agente.')); return; }
    draft = { ...a };
    const set = (k) => (e) => { draft[k] = e.target.value; };
    const clients = ctx.snap.clients || [];
    const client = clients.find((c) => c.id === draft.client);
    const models = h('datalist#models', (client?.models || []).map((m) => h('option', { value: m })));
    const others = ctx.snap.agents.filter((x) => x.id !== a.id);
    props.replaceChildren(
      h('div.panel.raised', h('div.panel-title', 'Propriedades'), h('div.panel-body',
        h('div.form',
          field('Nome:', h('input.field', { value: a.name, oninput: set('name') })),
          field('Função:', h('input.field', { value: a.role || '', oninput: set('role') })),
          field('Cliente:', h('select.field', { onchange: set('client') }, clients.map((c) => h('option', { value: c.id, selected: c.id === a.client }, `${c.name}${c.installed ? '' : ' (não instalado)'}`)))),
          field('Nível:', h('select.field', { onchange: set('tier') }, ctx.snap.tiers.map((t) => h('option', { value: t, selected: t === a.tier }, t)))),
          field('Modelo:', h('input.field', { value: a.model || '', placeholder: a.model_in_use || 'do nível', list: 'models', oninput: set('model') })), models,
          field('Superior:', h('select.field', { onchange: set('parent') }, h('option', { value: '' }, '—'), others.map((o) => h('option', { value: o.id, selected: o.id === a.parent }, o.name)))),
          field('Emblema:', h('div.row', emblem(a.emblem, 22), h('select.field.grow', { onchange: set('emblem') }, ctx.snap.emblems.map((e) => h('option', { value: e, selected: e === a.emblem }, e))))),
          field('Status:', h('div.row', dot(a.status), STATUS_LABEL[a.status]))),
        h('div', { style: { margin: '8px 0 3px' } }, 'Instruções do agente:'),
        h('textarea.field', { rows: 3, style: { width: '100%' }, value: a.instructions || '', placeholder: 'O que este agente sempre deve saber ou seguir', oninput: set('instructions') }),
        h('div', { style: { margin: '8px 0 3px' } }, 'Comando:'),
        h('div.cmd.sunken', `> ${a.command || ''}`),
        h('div.row', { style: { justifyContent: 'flex-end', marginTop: '8px' } }, h('button.btn.default', { onclick: save }, 'Aplicar')))),
      h('div.col',
        h('button.btn', { onclick: async () => { const f = await facePicker(a.face); if (f) { draft.face = f; save(); } } }, face(a.face, 22), 'Personalizar rostinho…'),
        h('button.btn', { onclick: () => ctx.go('console', a.id) }, 'Abrir Console'),
        h('button.btn', { onclick: () => ctx.call('TogglePause', a.id) }, a.paused ? 'Ativar' : 'Pausar'),
        h('button.btn', { onclick: () => { ctx.newParent = a.id; ctx.go('novo'); } }, 'Novo Subordinado…'),
        h('button.btn', { onclick: remove }, 'Remover')));
  }

  async function save() {
    const { status, model_in_use, client_name, client_color, last_line, command, ...agent } = draft;
    await ctx.call('UpdateAgent', agent);
    toast(`${agent.name} salvo.`);
  }
  async function remove() {
    const a = ctx.find(ctx.agent);
    const subs = ctx.snap.agents.filter((x) => x.parent === a.id).length;
    if (!(await confirmBox(`Remover ${a.name}?${subs ? `\nOs ${subs} subordinado(s) passam a responder ao superior dele.` : ''}`))) return;
    await ctx.call('RemoveAgent', a.id);
  }

  function draw() {
    const roots = (ctx.snap.agents || []).filter((a) => !a.parent);
    chart.replaceChildren(roots.length ? h('ul.tree', roots.map(node)) : h('div.empty', 'Nenhum agente. Use "Novo Agente".'));
    drawProps();
  }
  draw();
  return { el, update: draw };
}
