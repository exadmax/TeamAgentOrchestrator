// Mesa do Agente: the agent at its desk saying what it needs, and the task board.
// Responder / Delegar / Arquivar act on the selected task; with no task they talk to the agent freely.
import { api } from '../api.js';
import { h, icon, dialog, sheetUrl, toast } from '../ui.js';
import { answer, delegate, openTask } from './task.js';

// Card kind of an open task, in the prototype's categories.
function kindOf(t) {
  if (t.priority === 'high') return { title: 'Prioridade Alta', icon: 'alert', alert: true };
  if (t.question) return { title: 'Suporte', icon: 'headset' };
  if (t.status === 'review') return { title: 'Follow-up', icon: 'clip' };
  if (t.status === 'in_progress') return { title: 'Análise', icon: 'bars' };
  return { title: 'Nova Tarefa', icon: 'star' };
}

// What the agent says and the suggested answer, from the task state.
function speech(t, a) {
  if (!t) return { say: a?.status === 'ativo' ? 'Trabalhando nisso…' : `Olá! Sou ${a?.name || 'o agente'}. O que devo fazer?`,
    tip: 'escolha uma tarefa no quadro, ou use Responder para mandar uma instrução livre.' };
  if (t.running) return { say: `Trabalhando em ${t.id}…`, tip: 'acompanhe a saída na aba Console.' };
  if (t.question) return { say: t.question, tip: 'responda e o agente continua de onde parou, ou delegue a outro agente.', err: t.question.startsWith('falhou') };
  switch (t.status) {
    case 'proposed': return { say: `Proponho: ${t.title}. Posso seguir?`, tip: 'aprovar a proposta (Responder) ou arquivar.' };
    case 'ready': return { say: `Posso começar ${t.id}: ${t.title}?`, tip: 'Responder manda o agente começar agora.' };
    case 'in_progress': return { say: t.summary || `Estou com ${t.title}.`, tip: 'Responder manda o agente continuar.' };
    case 'review': return { say: `Terminei ${t.id}: ${t.summary || t.title}`, tip: 'Responder pede a revisão a um agente de outro cliente.' };
    default: return { say: `${t.id} concluída.`, tip: 'nada a fazer; arquive para tirar do quadro.' };
  }
}

export default function mesa(ctx) {
  let taskId = null;
  const title = h('div.panel-title');
  const bubble = h('div.bubble');
  const suggest = h('div.suggest.sunken');
  const board = h('div.panel-body', { style: { background: 'var(--face)' } });
  const count = h('span.right', { style: { color: '#000' } });
  const pick = h('select.field', { onchange: (e) => { ctx.agent = e.target.value; taskId = null; draw(); } });
  const el = h('div.page',
    h('div.col.grow', { style: { minWidth: 0 } },
      h('div.panel.raised', title, h('div', { style: { padding: '6px' } },
        h('div.scene.sunken', { style: { backgroundImage: `url(${sheetUrl})`, backgroundSize: '157.4% auto', backgroundPosition: '13.26% 45%' } }, bubble))),
      suggest,
      h('div.row', { style: { gap: '10px' } },
        h('button.btn.big.default.grow', { onclick: reply }, icon('reply', 24), 'Responder'),
        h('button.btn.big.grow', { onclick: del }, icon('delegate', 24), 'Delegar'),
        h('button.btn.big.grow', { onclick: archive }, icon('archive', 24), 'Arquivar'))),
    h('div.panel.raised', { style: { width: '260px', flex: 'none' } },
      h('div.panel-title', { style: { background: 'none', color: '#000' } }, icon('cube', 24), 'Quadro de Tarefas', count), board));

  const task = () => (ctx.board?.tasks || []).find((t) => t.id === taskId);

  async function reply() {
    const t = task();
    const a = ctx.find(ctx.agent);
    if (!t) {
      const text = await dialog('Responder', (close) => {
        const input = h('textarea.field', { rows: 4, style: { width: '420px' } });
        return [h('div.dialog-body.col', `Instrução para ${a?.name}:`, input),
          h('div.dialog-btns', h('button.btn.default', { onclick: () => close(input.value.trim()) }, 'Enviar'), h('button.btn', { onclick: () => close() }, 'Cancelar'))];
      });
      if (text) { await ctx.call('RunAgent', a.id, text); toast(`${a.name} recebeu a instrução.`); }
      return;
    }
    if (t.question) return answer(ctx, t);
    if (t.status === 'proposed') return ctx.call('ApproveTask', t.id).then(() => toast(`${t.id} aprovada.`));
    if (t.status === 'review') return ctx.call('ReviewTask', t.id).then(() => toast(`${t.id} foi para revisão.`));
    if (t.status === 'ready' || t.status === 'in_progress') return ctx.call('RunTask', t.id).then(() => toast(`${t.assignee_name} começou ${t.id}.`));
    return openTask(ctx, t.id);
  }
  async function del() {
    const t = task();
    if (t) return delegate(ctx, t);
    toast('Escolha uma tarefa no quadro para delegar.');
  }
  async function archive() {
    const t = task();
    if (!t) return;
    await ctx.call('ArchiveTask', t.id);
    taskId = null;
    toast(`${t.id} arquivada.`);
  }

  function draw() {
    const open = (ctx.board?.tasks || []).filter((t) => t.status !== 'done' && !t.archived)
      .sort((x, y) => (y.priority === 'high') - (x.priority === 'high') || !!y.question - !!x.question || x.id.localeCompare(y.id));
    if (taskId && !open.some((t) => t.id === taskId)) taskId = null;
    const t = task();
    if (t?.assignee) ctx.agent = t.assignee;
    const a = ctx.find(ctx.agent);
    pick.replaceChildren(...(ctx.snap.agents || []).map((x) => h('option', { value: x.id, selected: x.id === ctx.agent }, x.name)));
    title.replaceChildren(icon('agent', 20), `Agente: ${a?.name || '—'} (${a?.client_name || ''})${t ? ` · ${t.id}` : ''}`, h('span.right', pick));
    const s = speech(t, a);
    bubble.textContent = s.say.length > 150 ? `${s.say.slice(0, 149)}…` : s.say;
    bubble.style.color = s.err ? 'var(--err)' : '';
    suggest.replaceChildren(icon('bulb', 24), h('div', h('b', 'Resposta sugerida: '), s.tip));
    count.textContent = String(open.length);
    board.replaceChildren(...(open.length ? open.map((x) => {
      const k = kindOf(x);
      return h('div.taskcard.raised', { style: { outline: x.id === taskId ? '2px dotted #000' : '' }, onclick: () => { taskId = x.id; draw(); }, ondblclick: () => openTask(ctx, x.id) },
        h(`div.tc-title${k.alert ? '.alert' : ''}`, icon(k.icon, 18), h('span.grow', k.title),
          h('span', { title: 'Arquivar', style: { cursor: 'default' }, onclick: (e) => { e.stopPropagation(); ctx.call('ArchiveTask', x.id); } }, '✕')),
        h('div.tc-body', h('div', x.running ? '… ' : '', x.title), h('div.muted', `→ ${x.assignee_name || 'sem responsável'}`)));
    }) : [h('div.empty.small', 'Nenhuma tarefa aberta. Crie na aba Sprints.')]));
  }
  draw();
  return { el, update: draw, board: draw };
}
