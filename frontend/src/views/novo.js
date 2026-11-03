import { t as tr } from '../i18n/index.js';
// Novo Agente: three-step wizard (identity, client and model, face).
import { h, face, emblem, faceIds, toast } from '../ui.js';

export default function novo(ctx) {
  const firstInstalled = (ctx.snap.clients || []).find((c) => c.installed)?.id || 'claude';
  const a = { name: tr('wizard.new_agent'), role: tr('wizard.assistant'), parent: ctx.newParent || ctx.snap.agents?.[0]?.id || '', emblem: 'codigo',
    client: firstInstalled, tier: 'medium', model: '', face: 'av_p_robo', instructions: '' };
  ctx.newParent = null;
  let step = 1;
  const body = h('div.grow');
  const art = h('div.art.sunken');
  const back = h('button.btn', { onclick: () => { step--; draw(); } }, tr('wizard.back'));
  const next = h('button.btn.default', { onclick: advance }, tr('wizard.next'));
  const el = h('div.page', { style: { alignItems: 'center', justifyContent: 'center', background: 'var(--desk)' } },
    h('div.dialog.raised', h('div.titlebar', h('span.title', tr('wizard.new_agent_wizard'))),
      h('div.dialog-body', h('div.wizard', art, body)),
      h('div.dialog-btns', { style: { borderTop: '1px solid var(--shadow)', paddingTop: '10px', margin: '0 12px' } }, back, next,
        h('button.btn', { onclick: () => ctx.go('org') }, tr('common.cancel')))));

  const set = (k) => (e) => { a[k] = e.target.value; if (k === 'emblem' || k === 'client') draw(); };
  const row = (label, input) => [h('label', label), input];

  function draw() {
    art.replaceChildren(face(a.face, 120));
    back.disabled = step === 1;
    next.textContent = step === 3 ? tr('wizard.finish') : tr('wizard.next');
    const client = (ctx.snap.clients || []).find((c) => c.id === a.client);
    if (step === 1) {
      body.replaceChildren(h('h3', tr('wizard.step_1_of_3_identity')), h('p.small', tr('wizard.name_the_agent_and_choose_who_it')),
        h('div.form', { style: { gridTemplateColumns: '110px 1fr' } },
          row(tr('wizard.agent_name'), h('input.field', { value: a.name, oninput: set('name') })),
          row(tr('wizard.role'), h('input.field', { value: a.role, oninput: set('role') })),
          row(tr('wizard.reports_to'), h('select.field', { onchange: set('parent') }, h('option', { value: '' }, tr('wizard.nobody_top_level')),
            (ctx.snap.agents || []).map((x) => h('option', { value: x.id, selected: x.id === a.parent }, x.name)))),
          row(tr('wizard.emblem'), h('div.row', emblem(a.emblem, 24), h('select.field.grow', { onchange: set('emblem') },
            ctx.snap.emblems.map((e) => h('option', { value: e, selected: e === a.emblem }, e)))))));
    } else if (step === 2) {
      body.replaceChildren(h('h3', tr('wizard.step_2_of_3_client_and_model')), h('p.small', tr('wizard.the_agent_runs_through_the_selected_command')),
        h('div.form', { style: { gridTemplateColumns: '110px 1fr' } },
          row(tr('wizard.client'), h('select.field', { onchange: set('client') }, (ctx.snap.clients || []).map((c) =>
            h('option', { value: c.id, selected: c.id === a.client }, `${c.name}${c.installed ? ` ${c.version}` : tr('wizard.not_installed')}`)))),
          row(tr('wizard.tier'), h('select.field', { onchange: set('tier') }, ctx.snap.tiers.map((t) => h('option', { value: t, selected: t === a.tier },
            `${t} → ${client?.tiers?.[t] || tr('models.default_model_short')}`)))),
          row(tr('wizard.model'), h('input.field', { value: a.model, placeholder: tr('wizard.blank_tier_s_model'), list: 'wiz-models', oninput: set('model') })),
          h('datalist#wiz-models', (client?.models || []).map((m) => h('option', { value: m })))),
        h('div', { style: { margin: '8px 0 3px' } }, tr('wizard.instructions_optional')),
        h('textarea.field', { rows: 4, style: { width: '100%' }, value: a.instructions, oninput: set('instructions') }));
    } else {
      body.replaceChildren(h('h3', tr('wizard.step_3_of_3_face')), h('p.small', tr('wizard.choose_how_the_agent_looks_in_the')),
        h('div.faces.sunken', { style: { maxHeight: '230px' } }, faceIds.map((id) =>
          h(`div.opt${id === a.face ? '.sel' : ''}`, { onclick: () => { a.face = id; draw(); } }, face(id, 48)))));
    }
  }
  async function advance() {
    if (step === 1 && !a.name.trim()) return;
    if (step < 3) { step++; draw(); return; }
    await ctx.call('AddAgent', a);
    toast(tr('wizard.joined_the_team', { p1: a.name }));
    const created = ctx.snap.agents.at(-1);
    ctx.go('org', created?.id);
  }
  draw();
  return { el };
}
