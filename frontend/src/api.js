// Thin wrapper over the Wails bindings (window.go.main.App) and events (window.runtime).
// Outside Wails (vite dev in a normal browser) it falls back to a mock, so the screens can be
// designed without the Go side.
import { mock } from './mock.js';

const app = () => window.go?.main?.App;
export const inWails = () => !!app();

export const api = new Proxy({}, {
  get: (_, name) => (...args) => (app() ? app()[name](...args) : mock.call(name, ...args)),
});

export function on(event, fn) {
  if (window.runtime?.EventsOn) return window.runtime.EventsOn(event, fn);
  return mock.on(event, fn);
}
