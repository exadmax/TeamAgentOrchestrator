// Novo Agente: three-step wizard (identity, client and model, face).
import { h, face, emblem, faceIds, toast } from '../ui.js';

export default function novo(ctx) {
  const firstInstalled = (ctx.snap.clients || []).find((c) => c.installed)?.id || 'claude';
  const a = { name: 'Novo Agente', role: 'Auxiliar', parent: ctx.newParent || ctx.snap.agents?.[0]?.id || '', emblem: 'codigo',
    client: firstInstalled, tier: 'medium', model: '', face: 'av_p_robo', instructions: '' };
  ctx.newParent = null;
  let step = 1;
  const body = h('div.grow');
  const art = h('div.art.sunken');
  const back = h('button.btn', { onclick: () => { step--; draw(); } }, '< Voltar');
  const next = h('button.btn.default', { onclick: advance }, 'Avançar >');
  const el = h('div.page', { style: { alignItems: 'center', justifyContent: 'center', background: 'var(--desk)' } },
    h('div.dialog.raised', h('div.titlebar', h('span.title', 'Assistente de Novo Agente')),
      h('div.dialog-body', h('div.wizard', art, body)),
      h('div.dialog-btns', { style: { borderTop: '1px solid var(--shadow)', paddingTop: '10px', margin: '0 12px' } }, back, next,
        h('button.btn', { onclick: () => ctx.go('org') }, 'Cancelar'))));

  const set = (k) => (e) => { a[k] = e.target.value; if (k === 'emblem' || k === 'client') draw(); };
  const row = (label, input) => [h('label', label), input];

  function draw() {
    art.replaceChildren(face(a.face, 120));
    back.disabled = step === 1;
    next.textContent = step === 3 ? 'Concluir' : 'Avançar >';
    const client = (ctx.snap.clients || []).find((c) => c.id === a.client);
    if (step === 1) {
      body.replaceChildren(h('h3', 'Passo 1 de 3 - Identidade'), h('p.small', 'Dê um nome ao agente e escolha a quem ele responde no organograma.'),
        h('div.form', { style: { gridTemplateColumns: '110px 1fr' } },
          row('Nome do agente:', h('input.field', { value: a.name, oninput: set('name') })),
          row('Função:', h('input.field', { value: a.role, oninput: set('role') })),
          row('Responde a:', h('select.field', { onchange: set('parent') }, h('option', { value: '' }, '— ninguém (topo) —'),
            (ctx.snap.agents || []).map((x) => h('option', { value: x.id, selected: x.id === a.parent }, x.name)))),
          row('Emblema:', h('div.row', emblem(a.emblem, 24), h('select.field.grow', { onchange: set('emblem') },
            ctx.snap.emblems.map((e) => h('option', { value: e, selected: e === a.emblem }, e)))))));
    } else if (step === 2) {
      body.replaceChildren(h('h3', 'Passo 2 de 3 - Cliente e modelo'), h('p.small', 'O agente roda pelo cliente de linha de comando escolhido.'),
        h('div.form', { style: { gridTemplateColumns: '110px 1fr' } },
          row('Cliente:', h('select.field', { onchange: set('client') }, (ctx.snap.clients || []).map((c) =>
            h('option', { value: c.id, selected: c.id === a.client }, `${c.name}${c.installed ? ` ${c.version}` : ' (não instalado)'}`)))),
          row('Nível:', h('select.field', { onchange: set('tier') }, ctx.snap.tiers.map((t) => h('option', { value: t, selected: t === a.tier },
            `${t} → ${client?.tiers?.[t] || 'padrão'}`)))),
          row('Modelo:', h('input.field', { value: a.model, placeholder: 'vazio = o modelo do nível', list: 'wiz-models', oninput: set('model') })),
          h('datalist#wiz-models', (client?.models || []).map((m) => h('option', { value: m })))),
        h('div', { style: { margin: '8px 0 3px' } }, 'Instruções (opcional):'),
        h('textarea.field', { rows: 4, style: { width: '100%' }, value: a.instructions, oninput: set('instructions') }));
    } else {
      body.replaceChildren(h('h3', 'Passo 3 de 3 - Rostinho'), h('p.small', 'Escolha como o agente aparece no escritório.'),
        h('div.faces.sunken', { style: { maxHeight: '230px' } }, faceIds.map((id) =>
          h(`div.opt${id === a.face ? '.sel' : ''}`, { onclick: () => { a.face = id; draw(); } }, face(id, 48)))));
    }
  }
  async function advance() {
    if (step === 1 && !a.name.trim()) return;
    if (step < 3) { step++; draw(); return; }
    await ctx.call('AddAgent', a);
    toast(`${a.name} entrou no time.`);
    const created = ctx.snap.agents.at(-1);
    ctx.go('org', created?.id);
  }
  draw();
  return { el };
}
