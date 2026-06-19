import "@testing-library/jest-dom/vitest"

// Sigma (WebGL renderer) references WebGLRenderingContext and WebGL2RenderingContext
// at module evaluation time. jsdom does not implement WebGL, so we stub both
// to avoid crashes during import. The actual Sigma class is mocked per-test-file
// in GraphCanvas tests via vi.mock("sigma").
if (typeof WebGLRenderingContext === "undefined") {
  // @ts-expect-error — stub for jsdom
  globalThis.WebGLRenderingContext = class WebGLRenderingContext {}
}
if (typeof WebGL2RenderingContext === "undefined") {
  // @ts-expect-error — stub for jsdom
  globalThis.WebGL2RenderingContext = class WebGL2RenderingContext {}
}

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
