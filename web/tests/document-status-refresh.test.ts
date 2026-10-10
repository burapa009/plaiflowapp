import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { runInNewContext } from "node:vm";
import test from "node:test";
import ts from "typescript";

test("pending OCR refreshes the server, stops for completed forms, and keeps manual refresh usable", () => {
  const source = ts.transpileModule(readFileSync(new URL("../app/o/[organization]/documents/document-status-refresh.tsx", import.meta.url), "utf8"), { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX } }).outputText;
  for (const active of [false, true]) {
    let tick: (() => void) | undefined, cleanup: (() => void) | undefined;
    let refreshed = 0, cleared = 0;
    const document = { visibilityState: "visible" };
    const loaded = { exports: {} as { default: (props: object) => { props: { children: { props: { onClick: () => void } }[] } } } };
    const nativeRequire = createRequire(import.meta.url);
    runInNewContext(source, {
      module: loaded, exports: loaded.exports, document,
      window: { setInterval: (fn: () => void, ms: number) => { assert.equal(ms, 2000); tick = fn; return 1; }, clearInterval: () => { cleared++; } },
      require: (name: string) => name === "react" ? { useEffect: (fn: () => (() => void) | undefined) => { cleanup = fn(); }, useTransition: () => [false, (fn: () => void) => fn()] }
        : name === "next/navigation" ? { useRouter: () => ({ refresh: () => { refreshed++; } }) } : nativeRequire(name),
    });
    const control = loaded.exports.default({ active, manual: true });
    assert.equal(Boolean(tick), active, "completed forms must not be refreshed while editing");
    tick?.();
    assert.equal(refreshed, Number(active));
    document.visibilityState = "hidden";
    tick?.();
    assert.equal(refreshed, Number(active));
    control.props.children[1].props.onClick();
    assert.equal(refreshed, Number(active) + 1, "same-route refresh must bypass the cached Link navigation");
    cleanup?.();
    assert.equal(cleared, Number(active));
  }
});
