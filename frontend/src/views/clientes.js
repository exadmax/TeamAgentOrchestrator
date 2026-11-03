// Clientes: the agent CLIs found on this machine, their versions and a quick test prompt.
import { api } from '../api.js';
import { h, group, toast } from '../ui.js';
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
      group('Clientes pelo npm', h('div.sunken', { style: { background: '#fff', maxHeight: '190px', overflow: 'auto' } },
        h('table.table', h('thead', h('tr', ['Cliente', 'Pacote', 'Instalado', 'Última', 'Situação'].map((t) => h('th', t)))), installTable)),
      h('div.row', { style: { justifyContent: 'flex-end', marginTop: '6px' } },
        h('button.btn', { onclick: () => api.DetectClients().then(() => toast('Detecção concluída.')) }, 'Procurar instalados'),
        h('button.btn', { onclick: (e) => { e.target.disabled = true; api.CheckUpdates().finally(() => { e.target.disabled = false; }); } }, 'Verificar atualizações'))),
      group('Clientes registrados', h('div.sunken', { style: { background: '#fff', overflow: 'auto' } },
        h('table.table', h('thead', h('tr', ['Cliente', 'Executável', 'Versão', 'Saída', 'Modelos', 'Agentes'].map((t) => h('th', t)))), regTable))),
      h('p.small.muted', 'Instalar e atualizar pelo próprio TeamAgent chega no marco 4. Por enquanto: npm i -g <pacote>.')),
    h('div.panel.raised', { style: { width: '330px', flex: 'none' } }, h('div.panel-title', 'Cliente'), edit));

  function draw() {
    const cs = ctx.snap.clients || [];
    installTable.replaceChildren(...cs.map((c) => h('tr', h('td', h('div.row', h('span.dot', { style: { background: c.color, borderRadius: 0 } }), c.name)),
      h('td.mono', c.npm_package), h('td', c.version || '—'), h('td', c.latest || '—'),
      h('td', !c.installed ? 'não instalado' : c.latest && c.latest !== c.version ? h('b', 'atualização disponível') : '✓ instalado'))));
    regTable.replaceChildren(...cs.filter((c) => c.installed).map((c) => h(`tr${c.id === sel ? '.sel' : ''}`, { onclick: () => { sel = c.id; draw(); } },
      h('td', c.name), h('td.mono', c.exe), h('td', c.version), h('td', c.output), h('td', c.models?.length || 0),
      h('td', (ctx.snap.agents || []).filter((a) => a.client === c.id).length))));
    const c = cs.find((x) => x.id === sel);
    if (!c) { edit.replaceChildren(h('div.empty', 'Nenhum cliente instalado.')); return; }
    const model = h('select.field', h('option', { value: '' }, 'padrão do cliente'), (c.models || []).map((m) => h('option', { value: m }, m)));
    const prompt = h('input.field.grow', { value: 'Olá, você está pronto? Responda em uma linha.' });
    edit.replaceChildren(
      h('div.form', h('label', 'Nome:'), h('input.field', { value: c.name, readOnly: true }),
        h('label', 'Executável:'), h('input.field', { value: c.path || c.exe, readOnly: true, title: c.path }),
        h('label', 'Saída:'), h('div', c.output), h('label', 'Modelo:'), model),
      h('div', 'Modelos:'), h('div.cmd.sunken', (c.models || []).join(', ')),
      group('Prompt de teste', h('div.col', h('div.row', prompt, h('button.btn', { onclick: () => api.TestClient(c.id, model.value, prompt.value)
        .then(() => { out.replaceChildren(); }).catch((e) => out.replaceChildren(termLine({ kind: 'error', text: String(e) }))) }, 'Testar')), out)));
  }
  api.Console(TEST).then((ls) => out.replaceChildren(...(ls || []).map(termLine)));
  draw();
  return { el, update: draw, line(l) { if (l.agent === TEST) { out.append(termLine(l)); out.scrollTop = out.scrollHeight; } } };
}
