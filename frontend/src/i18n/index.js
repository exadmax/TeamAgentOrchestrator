// Interface language: English by default, Brazilian Portuguese optional. The backend keeps the choice
// (App.Language / App.SetLanguage) and sends a "language" event when it changes.
import en from './en.json';
import ptBR from './pt-BR.json';

const catalogs = { en, 'pt-BR': ptBR };
let lang = 'en';

export const LANGS = [['en', en['language.english']], ['pt-BR', ptBR['language.portuguese']]];

export function setLang(l) {
  if (catalogs[l]) lang = l;
  document.documentElement.lang = lang;
}
export const getLang = () => lang;

// t('tasks.count', { n: 3 }) -> "3 tasks". A key missing in the language falls back to English, then to the key.
export function t(key, params = {}) {
  const s = catalogs[lang][key] ?? en[key] ?? key;
  return s.replace(/\{(\w+)\}/g, (m, k) => (params[k] !== undefined ? String(params[k]) : m));
}

// Locale for dates and numbers.
export const locale = () => (lang === 'pt-BR' ? 'pt-BR' : 'en-US');

// Backend codes stay stable; unknown values remain readable.
export function codeLabel(area, code) {
  const key = area + '.' + code;
  const label = t(key);
  return label === key ? (code || '') : label;
}
