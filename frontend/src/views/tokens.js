// Tokens: usage summed from .tao/sessions.jsonl, per agent.
import { api } from '../api.js';
import { h, dot, clientTag, fmt } from '../ui.js';

export default function tokens(ctx) {
  let day = '';
  const stats = h('div.row', { style: { gap: '8px' } });
  const table = h('tbody');
  const el = h('div.page.col', { style: { flexDirection: 'column' } },
    stats,
    h('div.row', h('label.radio', h('input', { type: 'radio', name: 'tk', checked: true, onchange: () => { day = ''; load(); } }), 'Todos os dias'),
      h('label.radio', h('input', { type: 'radio', name: 'tk', onchange: () => { day = new Date().toLocaleDateString('sv-SE'); load(); } }), 'Hoje')),
    h('div.sunken', { style: { flex: 1, overflow: 'auto', background: '#fff' } }, h('table.table',
      h('thead', h('tr', ['Agente', 'Cliente', 'Modelo', 'Sessões', 'Entrada', 'Saída', 'Total', 'Custo', 'Participação']
        .map((t, i) => h(`th${i >= 3 && i <= 7 ? '.num' : ''}`, t)))), table)),
    h('div.small.muted', 'Contagem lida do JSON de uso de cada cliente (claude stream-json, codex --json, opencode --format json). Custo só quando o cliente informa.'));

  async function load() {
    const rows = (await api.Tokens(day)) || [];
    const sum = (k) => rows.reduce((s, r) => s + (r[k] || 0), 0);
    const stat = (label, v) => h('div.stat.sunken', label, h('b', v));
    stats.replaceChildren(stat('Total gasto', fmt(sum('total'))), stat('Entrada', fmt(sum('input'))), stat('Saída', fmt(sum('output'))),
      stat('Custo informado', `US$ ${sum('cost_usd').toFixed(2)}`));
    table.replaceChildren(...(rows.length ? rows.map((r) => h('tr',
      h('td', h('div.row', dot(r.status), r.name)), h('td', r.client ? clientTag(r.client, r.color) : '—'), h('td.mono', r.model || '—'),
      h('td.num', r.sessions), h('td.num', fmt(r.input)), h('td.num', fmt(r.output)), h('td.num', h('b', fmt(r.total))),
      h('td.num', r.cost_usd ? r.cost_usd.toFixed(3) : '—'),
      h('td', h('div.row', h('div.bar.grow', Array.from({ length: Math.max(1, Math.round(r.share * 30)) }, () => h('i'))), `${(r.share * 100).toFixed(1)}%`))))
      : [h('tr', h('td.empty', { colSpan: 9 }, 'Nenhuma sessão registrada ainda.'))]));
  }
  load();
  return { el, line: (l) => l.kind === 'end' && load() };
}
