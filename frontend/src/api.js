// Thin wrapper over the Wails v3 runtime: Go methods of main.App by name, and events.
// Outside Wails (vite dev in a normal browser) it falls back to a mock, so the screens can be
// designed without the Go side.
import { Application, Call, Events, Window } from '@wailsio/runtime';
import { mock } from './mock.js';

// The native bridge of the webview is there from the first script; Wails' own window._wails.environment
// arrives only later, and the runtime package defines window._wails.invoke itself even in a browser.
export const inWails = () => !!(window.chrome?.webview || window.webkit?.messageHandlers?.external || window._wails?.environment);

export const api = new Proxy({}, {
  get: (_, name) => (...args) => (inWails() ? Call.ByName(`main.App.${name}`, ...args) : mock.call(name, ...args)),
});

export function on(event, fn) {
  if (inWails()) return Events.On(event, (ev) => fn(ev.data));
  return mock.on(event, fn);
}

// Window controls of the Win95 title bar.
export const win = {
  minimise: () => inWails() && Window.Minimise(),
  toggleMaximise: () => inWails() && Window.ToggleMaximise(),
  quit: () => inWails() && Application.Quit(),
};
