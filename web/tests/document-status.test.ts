import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import test from "node:test";
import { runInNewContext } from "node:vm";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";

const source = readFileSync(new URL("../app/o/[organization]/documents/document-visuals.tsx", import.meta.url), "utf8");
const compiled = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX } }).outputText;
const loadedModule = { exports: {} as {
  DocumentStatus: (props: { status: string }) => React.ReactNode;
  DocumentSummary: (props: { filename: string; size: number; status: string }) => React.ReactNode;
} };
runInNewContext(compiled, { module: loadedModule, exports: loadedModule.exports, require: createRequire(import.meta.url) });

test("document status badges show Thai labels and distinct semantic states", () => {
  const statuses = [
    ["Available", "พร้อมใช้", "is-available"],
    ["Archived", "เก็บถาวร", "is-archived"],
    ["Trash", "อยู่ในถังขยะ", "is-danger"],
    ["Purged", "ลบถาวร", "is-danger"],
    ["Checking", "กำลังตรวจ", "is-checking"],
    ["Rejected", "ไม่รับเอกสาร", "is-danger"],
  ];
  for (const [status, label, tone] of statuses) {
    for (const html of [
      renderToStaticMarkup(loadedModule.exports.DocumentStatus({ status })),
      renderToStaticMarkup(loadedModule.exports.DocumentSummary({ filename: "ใบเสร็จ.pdf", size: 1024, status })),
    ]) {
      assert.ok(html.includes(label), status);
      assert.ok(html.includes(`document-status ${tone}`), status);
      assert.match(html, /<span aria-hidden="true">[^<]+<\/span>/);
    }
  }
  const detail = readFileSync(new URL("../app/o/[organization]/documents/[document]/page.tsx", import.meta.url), "utf8");
  assert.match(detail, /<DocumentStatus status=\{item\.status\} \/>/);
});
