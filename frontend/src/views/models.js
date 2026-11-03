import { t as tr, locale, codeLabel } from '../i18n/index.js';
// "Modelos e cotas" and "Fluxo git": the two settings windows of the Ferramentas menu.
import { api, on } from '../api.js';
import { h, dialog, toast, errText, alertBox, confirmBox, modelLabel, translatedNodes } from '../ui.js';

const left = (until) => {
  const s = Math.max(0, Math.round((new Date(until) - Date.now()) / 1000));
  return s >= 3600 ? `${Math.floor(s / 3600)}h${String(Math.floor((s % 3600) / 60)).padStart(2, '0')}m` : `${Math.floor(s / 60)}m${String(s % 60).padStart(2, '0')}s`;
};
const when = (iso) => (iso ? new Date(iso).toLocaleString(locale(), { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' }) : tr('models.until_removed'));

async function run(fn) {
  try { return await fn(); } catch (e) { alertBox(errText(e), tr('common.error')); return undefined; }
}

export function modelsWindow() {
  return dialog(tr('models.models_and_quotas'), (close) => {
    const body = h('div.dialog-body.col', { style: { width: '640px' } });
    let view = null;
    const timer = setInterval(() => view && draw(), 1000); // the clocks count down

    function draw() {
      const v = view;
      const client = h('select.field', v.clients.map((c) => h('option', { value: c.id }, c.name)));
      const model = h('select.field.grow');
      const fill = () => model.replaceChildren(...(v.clients.find((c) => c.id === client.value)?.models || []).map((m) => h('option', { value: m }, modelLabel(m))));
      client.onchange = fill;
      fill();
      const minutes = h('select.field', [['0', tr('models.until_removed')], ['60', tr('models.1_hour')], ['300', tr('models.5_hours')], ['1440', tr('models.1_day')]].map(([v2, l]) => h('option', { value: v2 }, l)));
      const task = h('input.field', { placeholder: tr('models.all'), style: { width: '70px' } });
      body.replaceChildren(
        h('label.radio', h('input', { type: 'checkbox', checked: v.auto, onchange: (e) => act(() => api.SetAutoSwitch(e.target.checked)) }),
          h('b', tr('models.auto_switch')), tr('models.when_a_model_runs_out_of_quota')),
        h('div', h('b', tr('models.quota_clocks')), tr('models.when_each_model_returns')),
        h('table.table', h('tr', h('th', tr('models.client')), h('th', tr('models.model')), h('th', tr('models.returns_in')), h('th', tr('models.reason')), h('th', '')),
          v.clocks.length ? v.clocks.map((c) => h('tr', h('td', c.client), h('td', modelLabel(c.model)), h('td', h('b', left(c.until)), ` (${when(c.until)})`),
            h('td', tr('models.clock_description', { reason: codeLabel('clock.reason', c.reason), source: codeLabel('clock.source', c.source) })), h('td', h('button.btn', { onclick: () => act(() => api.ClearClock(c.client, c.model)) }, tr('models.available_now')))))
            : h('tr', h('td', { colSpan: 5 }, h('span.muted', tr('models.all_models_have_quota'))))),
        h('div', h('b', tr('models.model_overrides')), tr('models.take_priority_over_the_agent_s_model')),
        h('table.table', h('tr', h('th', tr('models.client')), h('th', tr('models.task')), h('th', tr('models.model')), h('th', tr('models.until')), h('th', '')),
          v.overrides.length ? v.overrides.map((o) => h('tr', h('td', o.client), h('td', o.task || tr('models.all')), h('td', modelLabel(o.model)), h('td', when(o.until)),
            h('td', h('button.btn', { onclick: () => act(() => api.ClearModelOverride(o.key)) }, tr('common.remove')))))
            : h('tr', h('td', { colSpan: 5 }, h('span.muted', tr('models.no_overrides'))))),
        h('div.row', { style: { gap: '6px', flexWrap: 'wrap' } }, translatedNodes('models.override_form', { client, model, duration: minutes, task }),
          h('button.btn', { onclick: () => act(() => api.SetModelOverride(client.value, model.value, Number(minutes.value), task.value.trim())) }, tr('models.switch'))));
    }
    async function act(fn) {
      await run(fn);
      view = await api.Models();
      draw();
    }
    api.Models().then((v) => { view = v; draw(); });
    const off = on('models', (v) => { view = v; draw(); });
    const done = () => { clearInterval(timer); off?.(); close(); };
    return [body, h('div.dialog-btns', h('button.btn.default', { onclick: done }, tr('common.close')))];
  });
}

export function gitWindow(ctx) {
  return dialog(tr('git.git_workflow'), (close) => {
    const body = h('div.dialog-body.col', { style: { width: '560px' } });
    api.GitStatus().then((g) => {
      const on_ = h('input', { type: 'checkbox', checked: !g.off });
      const base = h('input.field', { value: g.base || '', placeholder: tr('git.origin_default') });
      const prefix = h('input.field', { value: g.prefix || 'tao/' });
      const repo = g.on || g.off;
      body.replaceChildren(
        repo ? h('div', translatedNodes('git.workflow_description', { worktree: h('code', '.tao/worktrees/<id>'),
          mode: g.gh && g.remote ? '' : h('b', tr('git.without_origin_or_gh_merges_are_local')) }))
          : h('div.suggest.sunken', g.reason || tr('git.the_project_folder_is_not_a_git')),
        h('div.form', { style: { gridTemplateColumns: '150px 1fr' } },
          h('label', tr('git.branch_per_task')), h('label.radio', on_, tr('git.on')),
          h('label', tr('git.base_branch')), base, h('label', tr('git.branch_prefix')), prefix,
          h('label', tr('git.github')), h('span', `${g.remote ? tr('git.origin') : tr('git.no_origin')} · ${g.gh ? tr('git.gh') : tr('git.gh_not_found')}`)),
        h('div.dialog-btns',
          h('button.btn.default', { disabled: !repo, onclick: async () => { if (await run(() => api.SetGit({ off: !on_.checked, base: base.value.trim(), prefix: prefix.value.trim() })) !== undefined) { toast(tr('git.git_workflow_saved_in_tao_json')); close(); } } }, tr('common.save')),
          h('button.btn', { onclick: () => close() }, tr('common.cancel'))));
    });
    return [body];
  });
}

export async function importCards(ctx) {
  const info = await api.CanImport();
  if (!info.available) { alertBox(tr('import.this_folder_has_no_docs_equipe_owners'), tr('import.import_cards')); return; }
  if (!(await confirmBox(tr('import.import_roles_from_docs_equipe_owners_json', { p1: info.has_tasks ? tr('import.the_current_team_and_board_tao_json') : '' }), tr('import.import_cards')))) return;
  const rep = await run(() => api.ImportCards());
  if (!rep) return;
  ctx.call('Snapshot');
  alertBox(tr('import.imported_agents_tasks_and_sprints', { p1: rep.agents, p2: rep.tasks, p3: rep.sprints, p4: rep.warnings?.length ? tr('import.warnings', { p1: rep.warnings.join('\n- ') }) : '' }), tr('import.import_cards'));
}
