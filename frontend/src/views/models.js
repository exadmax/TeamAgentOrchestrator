// "Modelos e cotas" and "Fluxo git": the two settings windows of the Ferramentas menu.
import { api, on } from '../api.js';
import { h, dialog, toast, errText, alertBox, confirmBox } from '../ui.js';

const left = (until) => {
  const s = Math.max(0, Math.round((new Date(until) - Date.now()) / 1000));
  return s >= 3600 ? `${Math.floor(s / 3600)}h${String(Math.floor((s % 3600) / 60)).padStart(2, '0')}m` : `${Math.floor(s / 60)}m${String(s % 60).padStart(2, '0')}s`;
};
const when = (iso) => (iso ? new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' }) : 'até remover');

async function run(fn) {
  try { return await fn(); } catch (e) { alertBox(errText(e), 'Erro'); return undefined; }
}

export function modelsWindow() {
  return dialog('Modelos e cotas', (close) => {
    const body = h('div.dialog-body.col', { style: { width: '640px' } });
    let view = null;
    const timer = setInterval(() => view && draw(), 1000); // the clocks count down

    function draw() {
      const v = view;
      const client = h('select.field', v.clients.map((c) => h('option', { value: c.id }, c.name)));
      const model = h('select.field.grow');
      const fill = () => model.replaceChildren(...(v.clients.find((c) => c.id === client.value)?.models || []).map((m) => h('option', { value: m }, m)));
      client.onchange = fill;
      fill();
      const minutes = h('select.field', [['0', 'até remover'], ['60', '1 hora'], ['300', '5 horas'], ['1440', '1 dia']].map(([v2, l]) => h('option', { value: v2 }, l)));
      const task = h('input.field', { placeholder: 'todas', style: { width: '70px' } });
      body.replaceChildren(
        h('label.radio', h('input', { type: 'checkbox', checked: v.auto, onchange: (e) => act(() => api.SetAutoSwitch(e.target.checked)) }),
          h('b', 'Troca automática: '), 'quando um modelo fica sem cota, a tarefa passa para outro modelo do mesmo cliente e, sem nenhum livre, para a reserva do agente.'),
        h('div', h('b', 'Relógios de cota'), ' (quando cada modelo volta):'),
        h('table.table', h('tr', h('th', 'Cliente'), h('th', 'Modelo'), h('th', 'Volta em'), h('th', 'Motivo'), h('th', '')),
          v.clocks.length ? v.clocks.map((c) => h('tr', h('td', c.client), h('td', c.model), h('td', h('b', left(c.until)), ` (${when(c.until)})`),
            h('td', `${c.reason} · ${c.source}`), h('td', h('button.btn', { onclick: () => act(() => api.ClearClock(c.client, c.model)) }, 'Já voltou'))))
            : h('tr', h('td', { colSpan: 5 }, h('span.muted', 'Nenhum modelo sem cota.')))),
        h('div', h('b', 'Trocas de modelo'), ' (valem antes do modelo do agente):'),
        h('table.table', h('tr', h('th', 'Cliente'), h('th', 'Tarefa'), h('th', 'Modelo'), h('th', 'Até'), h('th', '')),
          v.overrides.length ? v.overrides.map((o) => h('tr', h('td', o.client), h('td', o.task || 'todas'), h('td', o.model), h('td', when(o.until)),
            h('td', h('button.btn', { onclick: () => act(() => api.ClearModelOverride(o.key)) }, 'Remover'))))
            : h('tr', h('td', { colSpan: 5 }, h('span.muted', 'Nenhuma troca.')))),
        h('div.row', { style: { gap: '6px', flexWrap: 'wrap' } }, 'Trocar', client, 'para', model, 'por', minutes, 'na tarefa', task,
          h('button.btn', { onclick: () => act(() => api.SetModelOverride(client.value, model.value, Number(minutes.value), task.value.trim())) }, 'Trocar')));
    }
    async function act(fn) {
      await run(fn);
      view = await api.Models();
      draw();
    }
    api.Models().then((v) => { view = v; draw(); });
    const off = on('models', (v) => { view = v; draw(); });
    const done = () => { clearInterval(timer); off?.(); close(); };
    return [body, h('div.dialog-btns', h('button.btn.default', { onclick: done }, 'Fechar'))];
  });
}

export function gitWindow(ctx) {
  return dialog('Fluxo git', (close) => {
    const body = h('div.dialog-body.col', { style: { width: '560px' } });
    api.GitStatus().then((g) => {
      const on_ = h('input', { type: 'checkbox', checked: !g.off });
      const base = h('input.field', { value: g.base || '', placeholder: 'padrão do origin' });
      const prefix = h('input.field', { value: g.prefix || 'tao/' });
      const repo = g.on || g.off;
      body.replaceChildren(
        repo ? h('div', 'Cada tarefa ganha uma branch num worktree próprio (', h('code', '.tao/worktrees/<id>'), '): o agente nunca mexe na sua pasta. ',
          'Ao terminar, o orquestrador faz o commit, envia a branch e abre o PR', g.gh && g.remote ? '' : h('b', ' (sem origin ou sem gh: o merge será local)'),
          '. Depois da aprovação, "Fazer merge" junta na base.')
          : h('div.suggest.sunken', g.reason || 'A pasta do projeto não é um repositório git.'),
        h('div.form', { style: { gridTemplateColumns: '150px 1fr' } },
          h('label', 'Usar branch por tarefa:'), h('label.radio', on_, 'ligado'),
          h('label', 'Branch base:'), base, h('label', 'Prefixo das branches:'), prefix,
          h('label', 'GitHub:'), h('span', `${g.remote ? 'origin ✓' : 'sem origin'} · ${g.gh ? 'gh ✓' : 'gh não encontrado'}`)),
        h('div.dialog-btns',
          h('button.btn.default', { disabled: !repo, onclick: async () => { if (await run(() => api.SetGit({ off: !on_.checked, base: base.value.trim(), prefix: prefix.value.trim() })) !== undefined) { toast('Fluxo git salvo em tao.json.'); close(); } } }, 'Salvar'),
          h('button.btn', { onclick: () => close() }, 'Cancelar')));
    });
    return [body];
  });
}

export async function importCards(ctx) {
  const info = await api.CanImport();
  if (!info.available) { alertBox('Esta pasta não tem docs/equipe/owners.json: não há cartões para importar.', 'Importar cartões'); return; }
  if (!(await confirmBox(`Importar os papéis de docs/equipe/owners.json e os cartões de docs/tasks/?\n${info.has_tasks ? 'O time e o quadro atuais (tao.json e tao.tasks.json) serão substituídos.\n' : ''}Os cartões em si não são alterados.`, 'Importar cartões'))) return;
  const rep = await run(() => api.ImportCards());
  if (!rep) return;
  ctx.call('Snapshot');
  alertBox(`Importados ${rep.agents} agentes, ${rep.tasks} tarefas e ${rep.sprints} sprints.${rep.warnings?.length ? `\n\nAvisos:\n- ${rep.warnings.join('\n- ')}` : ''}`, 'Importar cartões');
}
