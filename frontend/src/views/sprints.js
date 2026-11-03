// Sprints: sprint list on the left; the selected sprint's activities, manual creation and generation by an agent.
import { h, group, dialog, confirmBox, toast } from '../ui.js';
import { openTask, statusDot } from './task.js';

export default function sprints(ctx) {
  const list = h('div.list.sunken', { style: { flex: 1 } });
  const head = h('div.panel-title');
  const goal = h('input.field.grow', { readOnly: true });
  const rows = h('tbody');
  const actions = h('div.row');
  const manualTitle = h('input.field.grow', { placeholder: 'Descreva a atividade', onkeydown: (e) => e.key === 'Enter' && addManual() });
  const manualWho = h('select.field');
  const genGoal = h('input.field.grow', { placeholder: 'Objetivo, ex.: portar o módulo de logs', onkeydown: (e) => e.key === 'Enter' && generate() });
  const genWho = h('select.field');
  const el = h('div.page',
    h('div.col', { style: { width: '220px', flex: 'none' } }, h('div', 'Sprints do projeto:'), list,
      h('button.btn', { onclick: newSprint }, 'Nova Sprint')),
    h('div.col.grow', { style: { minWidth: 0 } },
      h('div.panel.raised.grow', head,
        h('div.panel-body.col', { style: { padding: '6px' } },
          h('div.row', h('b', 'Objetivo:'), goal),
          h('div.sunken', { style: { flex: 1, overflow: 'auto', background: '#fff' } },
            h('table.table', h('thead', h('tr', ['ID', 'Atividade', 'Responsável', 'Status', 'Origem', ''].map((t) => h('th', t)))), rows)),
          actions)),
      h('div.row', { style: { alignItems: 'stretch' } },
        group('Nova atividade (manual)', h('div.col', manualTitle, h('div.row', manualWho, h('button.btn', { onclick: addManual }, 'Adicionar')))),
        group('Gerar com Agente', h('div.col', genGoal, h('div.row', genWho, h('button.btn', { onclick: generate }, 'Gerar')),
          h('div.small.muted', 'O agente quebra o objetivo em atividades e sugere responsáveis. Elas chegam como propostas.'))))));

  let sel = null; // selected sprint id ("" = backlog)
  const B = () => ctx.board || { sprints: [], tasks: [] };

  function draw() {
    const b = B();
    if (sel === null || (sel && !b.sprints.some((s) => s.id === sel))) sel = b.active || b.sprints.at(-1)?.id || '';
    list.replaceChildren(
      ...b.sprints.map((s) => h(`div.li${s.id === sel ? '.sel' : ''}`, { style: { display: 'block' }, onclick: () => { sel = s.id; draw(); } },
        h('b', `${s.id} — ${s.title}`), s.status === 'active' ? ' ●' : '',
        h('div.small', `${s.progress.done}/${s.progress.total} concluídas · ${({ planned: 'planejada', active: 'ativa', closed: 'fechada' })[s.status]}`))),
      h(`div.li${sel === '' ? '.sel' : ''}`, { style: { display: 'block' }, onclick: () => { sel = ''; draw(); } },
        h('b', 'Sem sprint'), h('div.small', `${b.backlog?.total || 0} atividades`)));
    const sp = b.sprints.find((s) => s.id === sel);
    head.replaceChildren(sp ? `Sprint ${sp.id.slice(1)} - ${sp.title}` : 'Atividades sem sprint',
      h('span.right', sp ? `${sp.progress.done}/${sp.progress.total} concluídas` : ''));
    goal.value = sp?.goal || '';
    const tasks = b.tasks.filter((t) => (t.sprint || '') === sel && !t.archived);
    rows.replaceChildren(...(tasks.length ? tasks.map((t) => h('tr', { ondblclick: () => openTask(ctx, t.id), title: 'Duplo clique para abrir', style: { cursor: 'default' } },
      h('td', h('b', t.id)), h('td', t.priority === 'high' ? h('b', { style: { color: 'var(--err)' } }, '! ') : '', t.title, t.question ? h('span.small', { style: { color: 'var(--err)' } }, '  (pergunta)') : ''),
      h('td', t.assignee_name || '—'), h('td', h('div.row', statusDot(t), t.running ? 'Rodando…' : t.status_label)),
      h('td', t.origin?.startsWith('agent:') ? h('span', { style: { color: 'var(--navy)', fontWeight: 'bold' } }, 'Agente') : t.origin?.startsWith('importado') ? 'Cartão' : 'Manual'),
      h('td', h('button.btn', { style: { minHeight: '18px', padding: '0 8px' }, onclick: () => openTask(ctx, t.id) }, 'Abrir…'))))
      : [h('tr', h('td.empty', { colSpan: 6 }, 'Nenhuma atividade. Crie à mão ou gere com um agente.'))]));
    actions.replaceChildren(...(sp ? [
      sp.status === 'planned' ? h('button.btn', { onclick: () => ctx.call('StartSprint', sp.id).then(() => toast(`${sp.id} iniciada.`)) }, 'Iniciar sprint') : null,
      sp.status !== 'closed' ? h('button.btn', { onclick: () => closeSprint(sp) }, 'Fechar sprint…') : null,
      sp.retro?.length ? h('span.small.muted', `Retro: ${sp.retro.join(' · ')}`) : null].filter(Boolean) : []));
    const opts = (s) => s.replaceChildren(...(ctx.snap.agents || []).map((a) => h('option', { value: a.id }, a.name)));
    const keepM = manualWho.value; const keepG = genWho.value;
    opts(manualWho); opts(genWho);
    manualWho.value = keepM || ctx.snap.agents?.[0]?.id || '';
    genWho.value = keepG || (ctx.snap.agents || []).find((a) => a.emblem === 'planejamento')?.id || ctx.snap.agents?.[0]?.id || '';
  }

  async function addManual() {
    if (!manualTitle.value.trim()) return;
    await ctx.call('AddTask', { title: manualTitle.value.trim(), assignee: manualWho.value, sprint: sel });
    manualTitle.value = '';
  }
  async function generate() {
    if (!genGoal.value.trim()) return;
    await ctx.call('GenerateTasks', genWho.value, genGoal.value.trim(), sel);
    toast(`${ctx.find(genWho.value)?.name} está planejando. As atividades chegam como propostas.`);
    genGoal.value = '';
  }
  async function newSprint() {
    const r = await dialog('Nova Sprint', (close) => {
      const title = h('input.field', { placeholder: 'Ex.: Porte do núcleo' });
      const g = h('input.field', { placeholder: 'O que estará pronto ao fim' });
      return [h('div.dialog-body.form', { style: { width: '420px' } }, h('label', 'Título:'), title, h('label', 'Objetivo:'), g),
        h('div.dialog-btns', h('button.btn.default', { onclick: () => close({ title: title.value.trim(), goal: g.value.trim() }) }, 'Criar'),
          h('button.btn', { onclick: () => close() }, 'Cancelar'))];
    });
    if (r?.title) { await ctx.call('NewSprint', r.title, r.goal); sel = null; }
  }
  async function closeSprint(sp) {
    const open = B().tasks.filter((t) => t.sprint === sp.id && t.status !== 'done' && !t.archived).length;
    const r = await dialog(`Fechar ${sp.id}`, (close) => {
      const retro = h('textarea.field', { rows: 4, style: { width: '420px' }, placeholder: 'Uma lição por linha' });
      const carry = h('select.field', h('option', { value: '' }, '— sem sprint —'),
        B().sprints.filter((s) => s.id !== sp.id && s.status !== 'closed').map((s) => h('option', { value: s.id }, `${s.id} — ${s.title}`)));
      return [h('div.dialog-body.col', 'Retrospectiva:', retro, open ? h('div', `${open} atividade(s) abertas vão para:`, carry) : null),
        h('div.dialog-btns', h('button.btn.default', { onclick: () => close({ retro: retro.value.split('\n').map((x) => x.trim()).filter(Boolean), carry: carry.value }) }, 'Fechar sprint'),
          h('button.btn', { onclick: () => close() }, 'Cancelar'))];
    });
    if (r && (await confirmBox(`Fechar ${sp.id} — ${sp.title}?`))) await ctx.call('CloseSprint', sp.id, r.retro, r.carry);
  }

  draw();
  return { el, update: draw, board: draw };
}
