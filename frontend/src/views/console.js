// Console: MS-DOS style prompt per agent, fed live by the CLI output.
import { api } from '../api.js';
import { h, dot, clientTag, hhmm } from '../ui.js';

export function termLine(l) {
  const text = l.kind === 'cmd' ? `C:\\TEAMAGENT> ${l.text}` : l.kind === 'tool' ? `» ${l.tool}${l.text ? ': ' + l.text : ''}`
    : l.kind === 'end' ? `■ ${l.text}` : l.kind === 'error' ? `! ${l.text}` : l.text;
  return h(`div.l-${l.kind}${l.error ? '.err' : ''}`, { title: hhmm(l.at) }, text);
}

export default function consoleView(ctx) {
  const list = h('div.list.sunken', { style: { width: '190px', flex: 'none' } });
  const term = h('div.term.sunken', { style: { flex: 1 } });
  const head = h('div.panel-title');
  const input = h('input.field.grow', { placeholder: 'Enviar instrução ao agente…', onkeydown: (e) => e.key === 'Enter' && send() });
  const sendBtn = h('button.btn', { onclick: send }, 'Enviar');
  const stopBtn = h('button.btn', { onclick: () => api.StopAgent(ctx.agent) }, 'Parar');
  const idle = h('div.small', { style: { color: 'var(--term-yellow)', background: '#000', padding: '0 10px' } });
  const el = h('div.page', list, h('div.panel.raised.grow', head, h('div.panel-body.col', { style: { padding: '4px' } }, term, idle,
    h('div.row', input, sendBtn, stopBtn))));

  async function send() {
    const text = input.value.trim();
    if (!text || !ctx.agent) return;
    await ctx.call('RunAgent', ctx.agent, text);
    input.value = '';
  }
  function drawList() {
    list.replaceChildren(...(ctx.snap.agents || []).map((a) => h(`div.li${a.id === ctx.agent ? '.sel' : ''}`,
      { onclick: () => { ctx.agent = a.id; load(); } }, dot(a.status), h('span.grow', a.name), clientTag(a.client_name, a.client_color))));
    const a = ctx.find(ctx.agent);
    head.replaceChildren(`MS-DOS Prompt - ${a?.name || ''}`, h('span.right', a ? `${a.client_name} · ${a.model_in_use || 'modelo padrão'}` : ''));
    const busy = a?.status === 'ativo';
    stopBtn.disabled = !busy;
    sendBtn.disabled = busy || a?.status === 'pausado';
    if (!busy) idle.textContent = '';
  }
  async function load() {
    drawList();
    const ls = ctx.agent ? await api.Console(ctx.agent) : [];
    term.replaceChildren(...(ls || []).map(termLine), h('span.cursor', '_'));
    term.scrollTop = term.scrollHeight;
    input.focus();
  }
  load();
  return {
    el,
    update: drawList,
    line(l) {
      if (l.agent !== ctx.agent) return;
      idle.textContent = '';
      term.lastChild?.remove();
      term.append(termLine(l), h('span.cursor', '_'));
      term.scrollTop = term.scrollHeight;
    },
    idle(e) {
      if (e.agent === ctx.agent) idle.textContent = `… aguardando resposta há ${e.seconds}s, sem saída (timeout em 300s)`;
    },
  };
}
