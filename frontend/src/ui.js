// Small DOM helpers and the Win95 building blocks shared by every screen.
import sheet from './assets/ui-sheet.png';

const faceFiles = import.meta.glob('./assets/avatars/*.png', { eager: true, query: '?url', import: 'default' });
export const faces = Object.fromEntries(Object.entries(faceFiles).map(([p, url]) => [p.split('/').pop().replace('.png', ''), url]));
export const faceIds = Object.keys(faces).filter((k) => /^av_(h|f|p)_?/.test(k) && !k.startsWith('av_e_')).sort();
export const emblemUrl = (e) => faces[`av_e_${e}`] || faces.av_e_ia;
export const faceUrl = (f) => faces[f] || faces.av_p_robo;

// h('div.cls#id', {attrs, onclick}, ...children)
export function h(tag, props, ...kids) {
  const [, name = 'div', rest = ''] = tag.match(/^([a-z0-9]*)(.*)$/i);
  const el = document.createElement(name || 'div');
  for (const m of rest.matchAll(/([.#])([\w-]+)/g)) m[1] === '.' ? el.classList.add(m[2]) : (el.id = m[2]);
  if (props && (typeof props !== 'object' || props instanceof Node || Array.isArray(props))) { kids.unshift(props); props = null; }
  for (const [k, v] of Object.entries(props || {})) {
    if (v == null || v === false) continue;
    if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else if (k === 'style' && typeof v === 'object') Object.assign(el.style, v);
    else if (k in el && k !== 'list') el[k] = v;
    else el.setAttribute(k, v === true ? '' : v);
  }
  for (const kid of kids.flat(Infinity)) if (kid != null && kid !== false) el.append(kid instanceof Node ? kid : String(kid));
  return el;
}

// Icons cut from the prototype sheet (positions in the sheet drawn at half size, 724x543).
const ICONS = {
  new: [20, 63], open: [63, 63], save: [107, 63], team: [166, 63], robot: [205, 63], tasks: [250, 63], chart: [299, 63],
  filter: [350, 63], search: [397, 63], help: [453, 63], agent: [32, 114], cube: [529, 111], bulb: [43, 397], reply: [70, 457],
  delegate: [232, 456], archive: [377, 456], star: [536, 151], clip: [538, 217], bars: [541, 284], headset: [541, 349], alert: [541, 414],
  app: [14, 6],
};
export function icon(name, size = 24) {
  const [x, y] = ICONS[name] || ICONS.robot;
  const f = size / 24;
  return h('span.sprite', { style: { width: `${size}px`, height: `${size}px`, backgroundImage: `url(${sheet})`,
    backgroundSize: `${724 * f}px ${543 * f}px`, backgroundPosition: `-${x * f}px -${y * f}px` } });
}
export const sheetUrl = sheet;

export const face = (id, size = 48) => h('img.face', { src: faceUrl(id), width: size, height: size, alt: '' });
export const emblem = (e, size = 20) => h('img.face', { src: emblemUrl(e), width: size, height: Math.round(size * 46 / 56), alt: '' });
export const dot = (status) => h(`span.dot.${status || 'ocioso'}`, { title: status });
export const clientTag = (name, color) => h('span.tag', { style: { background: color || '#555' } }, (name || '?').split(' ')[0].toLowerCase());

export const STATUS_LABEL = { ativo: 'Ativo', ocioso: 'Ocioso', erro: 'Erro', pausado: 'Pausado' };
export const fmt = (n) => Math.round(n || 0).toLocaleString('pt-BR');
export const hhmm = (iso) => (iso ? new Date(iso).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' }) : '');

export function panel(title, body, opts = {}) {
  return h('div.panel.raised', { style: opts.style },
    h('div.panel-title', opts.icon ? icon(opts.icon, 20) : null, title, opts.right ? h('span.right', opts.right) : null),
    h(`div.panel-body${opts.flush ? '' : ''}`, { style: opts.bodyStyle }, body));
}

export function group(legend, ...kids) {
  return h('div.group', h('span.legend', legend), ...kids);
}

// Modal Win95 dialog. Returns a promise with the value given to close().
export function dialog(title, build, { width } = {}) {
  return new Promise((resolve) => {
    const back = h('div.modal-back');
    const close = (v) => { back.remove(); resolve(v); };
    const box = h('div.dialog.raised', { style: { width } },
      h('div.titlebar', h('span.title', title), h('div.tb-btns', h('button.btn.tb-btn', { onclick: () => close(undefined) }, '✕'))));
    box.append(...[].concat(build(close)));
    back.append(box);
    document.body.append(back);
    back.querySelector('input,select,textarea,.btn.default')?.focus();
  });
}

export function alertBox(text, title = 'TeamAgent') {
  return dialog(title, (close) => [
    h('div.dialog-body.row', { style: { gap: '12px' } }, icon('alert', 24), h('div', { style: { whiteSpace: 'pre-wrap', maxWidth: '420px' } }, text)),
    h('div.dialog-btns', h('button.btn.default', { onclick: () => close() }, 'OK')),
  ]);
}

export function confirmBox(text, title = 'Confirmar') {
  return dialog(title, (close) => [
    h('div.dialog-body.row', { style: { gap: '12px' } }, icon('help', 24), h('div', text)),
    h('div.dialog-btns', h('button.btn.default', { onclick: () => close(true) }, 'Sim'), h('button.btn', { onclick: () => close(false) }, 'Não')),
  ]);
}

let toastTimer;
export function toast(text) {
  document.querySelector('.toast')?.remove();
  const el = h('div.toast.raised', h('div.titlebar', h('span.title', 'TeamAgent')), h('div.dialog-body', icon('bulb', 24), h('div', text)));
  document.body.append(el);
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.remove(), 3500);
}

export const errText = (e) => String(e?.message || e || 'erro');
