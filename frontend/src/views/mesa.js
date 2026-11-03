// Mesa do Agente: the selected agent at its desk, saying its last message, with Responder / Delegar / Arquivar.
// The task board on the right lists what each agent left pending (question or error); full tasks come in M2.
import { api } from '../api.js';
import { h, icon, dialog, sheetUrl, toast } from '../ui.js';

export default function mesa(ctx) {
  const said = {}; // agent -> last text / error
  const title = h('div.panel-title');
  const bubble = h('div.bubble');
  const suggest = h('div.suggest.sunken');
  const board = h('div.panel-body', { style: { background: 'var(--face)' } });
  const pick = h('select.field', { onchange: (e) => { ctx.agent = e.target.value; draw(); } });
  const el = h('div.page',
    h('div.col.grow', { style: { minWidth: 0 } },
      h('div.panel.raised', title, h('div', { style: { padding: '6px' } },
        h('div.scene.sunken', { style: { backgroundImage: `url(${sheetUrl})`, backgroundSize: '157.4% auto', backgroundPosition: '13.26% 45%' } }, bubble))),
      suggest,
      h('div.row', { style: { gap: '10px' } },
        h('button.btn.big.default.grow', { onclick: reply }, icon('reply', 24), 'Responder'),
        h('button.btn.big.grow', { onclick: delegate }, icon('delegate', 24), 'Delegar'),
        h('button.btn.big.grow', { onclick: () => { said[ctx.agent] = null; draw(); } }, icon('archive', 24), 'Arquivar'))),
    h('div.panel.raised', { style: { width: '250px', flex: 'none' } }, h('div.panel-title', { style: { background: 'none', color: '#000' } }, icon('cube', 24), 'Quadro de Tarefas'), board));

  async function ask(question, label) {
    return dialog(label, (close) => {
      const input = h('textarea.field', { rows: 4, style: { width: '420px' }, placeholder: question });
      return [h('div.dialog-body.col', h('div', question), input),
        h('div.dialog-btns', h('button.btn.default', { onclick: () => close(input.value.trim()) }, 'Enviar'), h('button.btn', { onclick: () => close() }, 'Cancelar'))];
    });
  }
  async function reply() {
    const a = ctx.find(ctx.agent);
    if (!a) return;
    const text = await ask(`Instrução para ${a.name}:`, 'Responder');
    if (text) { await ctx.call('RunAgent', a.id, said[a.id] ? `Sobre "${said[a.id].text}": ${text}` : text); toast(`${a.name} recebeu a instrução.`); }
  }
  async function delegate() {
    const a = ctx.find(ctx.agent);
    const others = ctx.snap.agents.filter((x) => x.id !== a?.id && x.status !== 'pausado');
    const to = await dialog('Delegar', (close) => {
      const sel = h('select.field', others.map((o) => h('option', { value: o.id }, `${o.name} (${o.role})`)));
      const note = h('textarea.field', { rows: 3, style: { width: '380px' }, value: said[a?.id]?.text || '' });
      return [h('div.dialog-body.col', 'Passar para:', sel, 'Instrução:', note),
        h('div.dialog-btns', h('button.btn.default', { onclick: () => close({ id: sel.value, text: note.value.trim() }) }, 'Delegar'), h('button.btn', { onclick: () => close() }, 'Cancelar'))];
    });
    if (to?.id && to.text) { await ctx.call('RunAgent', to.id, `${a.name} delegou para você: ${to.text}`); toast('Delegado.'); }
  }

  function draw() {
    const agents = ctx.snap.agents || [];
    const a = ctx.find(ctx.agent);
    pick.replaceChildren(...agents.map((x) => h('option', { value: x.id, selected: x.id === ctx.agent }, x.name)));
    title.replaceChildren(icon('agent', 20), `Agente: ${a?.name || '—'} (${a?.client_name || ''})`, h('span.right', pick));
    const last = said[ctx.agent];
    bubble.textContent = a?.status === 'ativo' ? 'Trabalhando nisso…' : last?.text ? last.text.slice(0, 140) : `Olá! Sou ${a?.name || 'o agente'}. O que devo fazer?`;
    bubble.style.color = last?.error ? 'var(--err)' : '';
    suggest.replaceChildren(icon('bulb', 24), h('div', h('b', 'Dica: '), last?.error ? 'o agente falhou; responda com uma correção ou delegue a outro agente.'
      : 'use Responder para mandar a próxima instrução, ou Delegar para passar a mensagem a outro agente.'));
    const pend = agents.filter((x) => said[x.id]);
    board.replaceChildren(...(pend.length ? pend.map((x) => h('div.taskcard.raised', { onclick: () => { ctx.agent = x.id; draw(); } },
      h(`div.tc-title${said[x.id].error ? '.alert' : ''}`, icon(said[x.id].error ? 'alert' : 'star', 18), said[x.id].error ? 'Erro' : 'Mensagem'),
      h('div.tc-body', said[x.id].text.slice(0, 80), h('div.muted', `→ ${x.name}`))))
      : [h('div.empty.small', 'Sem pendências. As tarefas e sprints chegam no marco 2.')]));
  }

  Promise.all((ctx.snap.agents || []).map((a) => api.Console(a.id).then((ls) => {
    const l = (ls || []).filter((x) => x.kind === 'text' || x.error).at(-1);
    if (l) said[a.id] = { text: l.text, error: l.error };
  }))).then(draw);
  draw();
  return { el, update: draw, line(l) { if (l.kind === 'text' || l.error) { said[l.agent] = { text: l.text, error: l.error }; draw(); } } };
}
