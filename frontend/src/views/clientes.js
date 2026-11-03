import { t as tr } from '../i18n/index.js';
// Clientes: the agent CLIs found on this machine, their versions and a quick test prompt.
import { api } from '../api.js';
import { h, group, toast, modelLabel } from '../ui.js';
import { termLine } from './console.js';

const TEST = '__test__';

export default function clientes(ctx) {
  let sel = (ctx.snap.clients || []).find((c) => c.installed)?.id || 'claude';
  const installTable = h('tbody');
  const regTable = h('tbody');
  const edit = h('div.panel-body.col');
  const out = h('div.term.sunken', { style: { flex: 1, minHeight: '120px', fontSize: '15px' } });
  const el = h('div.page',
    h('div.col.grow', { style: { minWidth: 0 } },
      group(tr('clients.npm_clients'), h('div.sunken', { style: { background: '#fff', maxHeight: '190px', overflow: 'auto' } },
        h('table.table', h('thead', h('tr', [tr('clients.client'), tr('clients.package'), tr('clients.installed'), tr('clients.latest'), tr('clients.status')].map((t) => h('th', t)))), installTable)),
      h('div.row', { style: { justifyContent: 'flex-end', marginTop: '6px' } },
        h('button.btn', { onclick: () => api.DetectClients().then(() => toast(tr('clients.detection_complete'))) }, tr('clients.find_installed')),
        h('button.btn', { onclick: (e) => { e.target.disabled = true; api.CheckUpdates().finally(() => { e.target.disabled = false; }); } }, tr('clients.check_for_updates')))),
      group(tr('clients.registered_clients'), h('div.sunken', { style: { background: '#fff', overflow: 'auto' } },
        h('table.table', h('thead', h('tr', [tr('clients.client'), tr('clients.executable'), tr('clients.version'), tr('clients.output'), tr('clients.models'), tr('clients.agents')].map((t) => h('th', t)))), regTable))),
      h('p.small.muted', tr('clients.install_and_update_support_arrives_in_milestone'))),
    h('div.panel.raised', { style: { width: '330px', flex: 'none' } }, h('div.panel-title', tr('clients.client')), edit));

  function draw() {
    const cs = ctx.snap.clients || [];
    installTable.replaceChildren(...cs.map((c) => h('tr', h('td', h('div.row', h('span.dot', { style: { background: c.color, borderRadius: 0 } }), c.name)),
      h('td.mono', c.npm_package), h('td', c.version || '—'), h('td', c.latest || '—'),
      h('td', !c.installed ? tr('clients.not_installed') : c.latest && c.latest !== c.version ? h('b', tr('clients.update_available')) : tr('clients.installed_2')))));
    regTable.replaceChildren(...cs.filter((c) => c.installed).map((c) => h(`tr${c.id === sel ? '.sel' : ''}`, { onclick: () => { sel = c.id; draw(); } },
      h('td', c.name), h('td.mono', c.exe), h('td', c.version), h('td', c.output), h('td', c.models?.length || 0),
      h('td', (ctx.snap.agents || []).filter((a) => a.client === c.id).length))));
    const c = cs.find((x) => x.id === sel);
    if (!c) { edit.replaceChildren(h('div.empty', tr('clients.no_clients_installed'))); return; }
    const model = h('select.field', h('option', { value: '' }, tr('clients.client_default')), (c.models || []).map((m) => h('option', { value: m }, modelLabel(m))));
    const prompt = h('input.field.grow', { value: tr('clients.hello_are_you_ready_reply_in_one') });
    edit.replaceChildren(
      h('div.form', h('label', tr('clients.name')), h('input.field', { value: c.name, readOnly: true }),
        h('label', tr('clients.executable_2')), h('input.field', { value: c.path || c.exe, readOnly: true, title: c.path }),
        h('label', tr('clients.output_2')), h('div', c.output), h('label', tr('clients.model')), model),
      h('div', tr('clients.models_2')), h('div.cmd.sunken', (c.models || []).map(modelLabel).join(', ')),
      group(tr('clients.test_prompt'), h('div.col', h('div.row', prompt, h('button.btn', { onclick: () => api.TestClient(c.id, model.value, prompt.value)
        .then(() => { out.replaceChildren(); }).catch((e) => out.replaceChildren(termLine({ kind: 'error', text: String(e) }))) }, tr('clients.test'))), out)));
  }
  api.Console(TEST).then((ls) => out.replaceChildren(...(ls || []).map(termLine)));
  draw();
  return { el, update: draw, line(l) { if (l.agent === TEST) { out.append(termLine(l)); out.scrollTop = out.scrollHeight; } } };
}
