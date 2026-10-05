import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { runInNewContext } from "node:vm";
import test from "node:test";
import ts from "typescript";

const compiled = ts.transpileModule(readFileSync(new URL("../app/o/[organization]/documents/upload-form.tsx", import.meta.url), "utf8"), {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX },
}).outputText;

function setup(files: File[], fetcher: typeof fetch, cookie = "__Host-plaiflow-csrf=test") {
  let resets = 0, refreshes = 0;
  const messages: unknown[] = [];
  const loaded = { exports: {} as { UploadForm: (props: { organization: string }) => { props: { onSubmit: (event: unknown) => Promise<void> } } } };
  const nativeRequire = createRequire(import.meta.url);
  runInNewContext(compiled, {
    module: loaded, exports: loaded.exports, File, FormData, crypto, AbortSignal, fetch: fetcher,
    document: { cookie },
    require: (name: string) => name === "react" ? {
      useRef: (value: unknown) => ({ current: value }),
      useState: (initial: unknown) => { let value = initial; return [value, (next: unknown) => { value = typeof next === "function" ? next(value) : next; messages.push(value); }]; },
    } : name === "next/navigation" ? { useRouter: () => ({ refresh: () => refreshes++ }) }
      : name === "./document-visuals" ? { DocumentIcon: () => null } : nativeRequire(name),
  });
  const form = loaded.exports.UploadForm({ organization: "team" });
  const event = { preventDefault() {}, currentTarget: { elements: { namedItem: () => ({ files }) }, reset: () => resets++ } };
  return { submit: () => form.props.onSubmit(event), state: () => ({ resets, refreshes, messages }) };
}

test("multi-file upload is sequential, CSRF protected and reuses retry keys", async () => {
  const calls: RequestInit[] = [];
  let active = 0;
  const files = [new File(["pdf"], "a.pdf", { type: "application/pdf" }), new File(["png"], "b.png", { type: "image/png" })];
  const harness = setup(files, async (_url, options) => {
    assert.equal(active++, 0);
    await Promise.resolve();
    active--;
    calls.push(options!);
    return new Response(JSON.stringify({}), { status: calls.length === 2 ? 503 : 200 });
  });
  await Promise.all([harness.submit(), harness.submit()]);
  assert.equal(calls.length, 2, JSON.stringify(harness.state()));
  assert.equal(harness.state().resets, 0, "retain files after partial failure");
  assert.equal(harness.state().refreshes, 1);
  await harness.submit();
  assert.equal(calls.length, 4);
  for (let i = 0; i < 2; i++) {
    const first = calls[i].headers as Record<string, string>;
    const retry = calls[i + 2].headers as Record<string, string>;
    assert.equal(first["X-CSRF-Token"], "test");
    assert.equal(first["Idempotency-Key"], retry["Idempotency-Key"]);
    assert.equal((calls[i].body as FormData).get("file") instanceof File, true);
  }
  assert.equal(harness.state().resets, 1);
});

test("missing CSRF, unsupported files and oversized batches never send requests", async () => {
  const file = new File(["pdf"], "a.pdf", { type: "application/pdf" });
  let calls = 0;
  const fetcher: typeof fetch = async () => { calls++; return new Response("{}"); };
  await setup([file], fetcher, "").submit();
  await setup(Array(51).fill(file), fetcher).submit();
  await setup([new File(["html"], "evil.html", { type: "text/html" })], fetcher).submit();
  await setup([new File(["webp"], "a.webp", { type: "image/webp" })], fetcher).submit();
  assert.equal(calls, 0);
});

test("lost access stops remaining uploads", async () => {
  const file = new File(["pdf"], "a.pdf", { type: "application/pdf" });
  let calls = 0;
  const harness = setup([file, file], async () => { calls++; return new Response("{}", { status: 403 }); });
  await harness.submit();
  assert.equal(calls, 1);
  assert.equal(harness.state().resets, 0);
});


test("quota exhaustion stops the batch and keeps files for retry", async () => {
  const file = new File(["pdf"], "a.pdf", { type: "application/pdf" });
  let calls = 0;
  const harness = setup([file, file], async () => {
    calls++;
    return new Response(JSON.stringify({ code: "document_quota_exhausted" }), { status: 409 });
  });
  await harness.submit();
  assert.equal(calls, 1);
  assert.equal(harness.state().resets, 0);
  assert.equal(harness.state().refreshes, 0);
});
