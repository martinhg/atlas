import "@testing-library/jest-dom/vitest"

// Node 26 exposes localStorage as undefined (experimental flag required).
// Vitest's populateGlobal skips keys already present in Node global, so jsdom's
// localStorage never lands on globalThis. Fix: grab it directly from jsdom's window.
declare const jsdom: { window: { localStorage: Storage; sessionStorage: Storage } }
if (typeof jsdom !== "undefined" && typeof localStorage === "undefined") {
  const _ls = jsdom.window.localStorage
  const _ss = jsdom.window.sessionStorage
  Object.defineProperty(globalThis, "localStorage", {
    value: _ls,
    writable: true,
    configurable: true,
  })
  Object.defineProperty(globalThis, "sessionStorage", {
    value: _ss,
    writable: true,
    configurable: true,
  })
}
