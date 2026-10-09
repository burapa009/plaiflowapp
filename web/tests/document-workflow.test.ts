import assert from "node:assert/strict";
import test from "node:test";
import { documentWorkflow, expenseDocumentTypes, expenseTypeOptions } from "../lib/document-workflow.ts";
import { formConfig, formLayout } from "../lib/document-form.ts";

test("workflow separates file acceptance, OCR, human review, and current confirmation", () => {
  assert.equal(documentWorkflow("Checking"), "waiting");
  assert.equal(documentWorkflow("Available", "NotScheduled"), "waiting");
  assert.equal(documentWorkflow("Available", "Running"), "processing");
  assert.equal(documentWorkflow("Available", "Queued"), "processing");
  assert.equal(documentWorkflow("Available", "Completed"), "unavailable");
  assert.equal(documentWorkflow("Available", "Completed", {status:"Draft",source_superseded:false}), "review");
  assert.equal(documentWorkflow("Available", "Completed", {status:"Confirmed",source_superseded:false}), "confirmed");
  assert.equal(documentWorkflow("Available", "Completed", {status:"Confirmed",source_superseded:true}), "review");
  assert.equal(documentWorkflow("Available", "Failed", {status:"Confirmed",source_superseded:false}), "error");
  assert.equal(documentWorkflow("Rejected"), "error");
  assert.equal(documentWorkflow("Available"), "unavailable");
  assert.equal(documentWorkflow("Trash", "Completed", {status:"Confirmed",source_superseded:false}), "inactive");
});
test("expense picker offers seven existing forms and preserves a legacy type", () => {
  assert.equal(expenseDocumentTypes.length, 7);
  assert.deepEqual(expenseTypeOptions("receipt"), expenseDocumentTypes);
  assert.deepEqual(expenseTypeOptions("bank_statement"), ["bank_statement", ...expenseDocumentTypes]);
  for (const type of expenseDocumentTypes) assert.ok(formConfig.types[type]);
  const layout = (type:string) => formLayout({version:1,type,fields:{},tables:{}}).map(item=>item.key);
  assert.ok(layout("invoice").includes("credit"));
  assert.ok(!layout("invoice").includes("payment"));
  assert.ok(layout("receipt").includes("payment"));
  assert.ok(layout("receipt_substitute").includes("substitute"));
});
