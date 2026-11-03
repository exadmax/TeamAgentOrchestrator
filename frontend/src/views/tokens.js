import { t as tr, locale } from '../i18n/index.js';
// Tokens: usage summed from .tao/sessions.jsonl, per agent.
import { api } from '../api.js';
import { h, dot, clientTag, fmt, modelLabel } from '../ui.js';

export default function tokens(ctx) {
  let day = '';
  const stats = h('div.row', { style: { gap: '8px' } });
  const table = h('tbody');
  const el = h('div.page.col', { style: { flexDirection: 'column' } },
    stats,
    h('div.row', h('label.radio', h('input', { type: 'radio', name: 'tk', checked: true, onchange: () => { day = ''; load(); } }), tr('tokens.all_days')),
      h('label.radio', h('input', { type: 'radio', name: 'tk', onchange: () => { day = new Date().toLocaleDateString('sv-SE'); load(); } }), tr('tokens.today'))),
    h('div.sunken', { style: { flex: 1, overflow: 'auto', background: '#fff' } }, h('table.table',
      h('thead', h('tr', [tr('tokens.agent'), tr('tokens.client'), tr('tokens.model'), tr('tokens.sessions'), tr('tokens.input'), tr('tokens.output'), tr('tokens.total'), tr('tokens.cost'), tr('tokens.share')]
        .map((t, i) => h(`th${i >= 3 && i <= 7 ? '.num' : ''}`, t)))), table)),
    h('div.small.muted', tr('tokens.usage_comes_from_each_client_s_json')));

  async function load() {
    const rows = (await api.Tokens(day)) || [];
    const sum = (k) => rows.reduce((s, r) => s + (r[k] || 0), 0);
    const stat = (label, v) => h('div.stat.sunken', label, h('b', v));
    stats.replaceChildren(stat(tr('tokens.total_used'), fmt(sum('total'))), stat(tr('tokens.input'), fmt(sum('input'))), stat(tr('tokens.output'), fmt(sum('output'))),
      stat(tr('tokens.reported_cost'), `US$ ${sum('cost_usd').toLocaleString(locale(), { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`));
    table.replaceChildren(...(rows.length ? rows.map((r) => h('tr',
      h('td', h('div.row', dot(r.status), r.name)), h('td', r.client ? clientTag(r.client, r.color) : '—'), h('td.mono', r.model ? modelLabel(r.model) : '—'),
      h('td.num', r.sessions), h('td.num', fmt(r.input)), h('td.num', fmt(r.output)), h('td.num', h('b', fmt(r.total))),
      h('td.num', r.cost_usd ? r.cost_usd.toLocaleString(locale(), { minimumFractionDigits: 3, maximumFractionDigits: 3 }) : '—'),
      h('td', h('div.row', h('div.bar.grow', Array.from({ length: Math.max(1, Math.round(r.share * 30)) }, () => h('i'))), `${(r.share * 100).toLocaleString(locale(), { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%`))))
      : [h('tr', h('td.empty', { colSpan: 9 }, tr('tokens.no_sessions_recorded_yet')))]));
  }
  load();
  return { el, line: (l) => l.kind === 'end' && load() };
}
