import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { runInNewContext } from "node:vm";
import test from "node:test";
import ts from "typescript";

test("embedded review guards inbox forms and retains edits when exit is cancelled", () => {
  const compiled = ts.transpileModule(readFileSync(new URL("../app/o/[organization]/documents/[document]/review-workspace.tsx", import.meta.url), "utf8"), {
    compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX },
  }).outputText;
  const listeners = new Map<string, (event: unknown) => void>();
  let prompts = 0, prevented = 0, stopped = 0;
  let allowExit = false;
  class Form {
    private review: boolean;
    constructor(review = false) { this.review = review; }
    matches() { return this.review; }
    closest() { return null; }
  }
  const loaded = { exports: {} as { default: (props: object) => { props: { onInputCapture: (event: unknown) => void } } } };
  const nativeRequire = createRequire(import.meta.url);
  runInNewContext(compiled, {
    module: loaded, exports: loaded.exports, HTMLFormElement: Form,
    document: { addEventListener: (name: string, handler: (event: unknown) => void) => listeners.set(name, handler) },
    window: { addEventListener() {}, confirm: () => { prompts++; return allowExit; } },
    require: (name: string) => name === "react" ? {
      useRef: (value: unknown) => ({ current: value }), useState: (value: unknown) => [value, () => {}], useEffect: (effect: () => void) => effect(),
    } : name.endsWith(".css") ? {} : nativeRequire(name),
  });
  const workspace = loaded.exports.default({ viewer: null, form: null });
  const submit = (review = false) => listeners.get("submit")!({ target: new Form(review), preventDefault: () => prevented++, stopPropagation: () => stopped++ });
  submit();
  assert.equal(prompts, 0);
  workspace.props.onInputCapture({ target: { closest: () => ({}) } });
  submit(true);
  assert.equal(prompts, 0, "review save itself must remain usable");
  submit();
  submit();
  assert.equal(prompts, 2, "cancelled exit retains the dirty state");
  assert.equal(prevented, 2);
  assert.equal(stopped, 2);
  allowExit = true;
  submit();
  submit();
  assert.equal(prompts, 3, "approved exit clears dirty state");
  assert.equal(prevented, 2);
});
