// Sprints: placeholder of milestone 2 (generic tasks and sprints replacing card.py).
import { h, icon, panel } from '../ui.js';

export default function sprints(ctx) {
  const el = h('div.page', { style: { alignItems: 'center', justifyContent: 'center' } },
    panel('Sprints - em construção (marco 2)', h('div.row', { style: { gap: '14px', alignItems: 'flex-start', maxWidth: '520px' } }, icon('tasks', 48),
      h('div', h('p', 'Aqui ficam as sprints e as atividades do projeto: criar à mão ou pedir a um agente para quebrar um objetivo em atividades, atribuir responsáveis e acompanhar cada etapa (começar, trabalho, submeter, PR, revisão, merge).'),
        h('p.muted', 'O núcleo já existe em Go (máquina de etapas em internal/store). A tela e o formato das tarefas vêm no marco 2.'),
        h('button.btn', { onclick: () => ctx.go('console') }, 'Enquanto isso, falar com um agente…'))), { style: { width: '620px' } }));
  return { el };
}
