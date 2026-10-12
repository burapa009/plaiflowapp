import test from "node:test";
import assert from "node:assert/strict";
import { fieldEvidence, formFieldLabel, visibleFormFields, type FormEvidence } from "../lib/document-evidence.ts";
import { expenseDocumentTypes } from "../lib/document-workflow.ts";
import type { CanonicalDocument } from "../lib/document-form.ts";

const data: CanonicalDocument = {version:1,type:"receipt",fields:{total_amount:"1070.00"},tables:{},source:{ocr_job_id:"run-1",fields:{total_amount:"1070.00"},tables:{}}};
const evidence: FormEvidence = {jobId:"run-1",fields:{total_amount:{presence:"found",normalized:"1070.00",raw:"รวม 1,070.00",confidence:"unrated",evidence:[{page:1,line:8,ocr_confidence:.95}]}}};
test("evidence preserves actual text, locations, unknowns and human corrections", () => {
  assert.equal(fieldEvidence(data,"total_amount",evidence).status,"อ่านจากเอกสาร");
  assert.equal(fieldEvidence(data,"total_amount",evidence).locations[0].line,8);
  assert.equal(fieldEvidence(data,"seller_tax_id",evidence).status,"ยังไม่มีข้อมูล");
  const edited = {...data,fields:{total_amount:""}};
  assert.equal(fieldEvidence(edited,"total_amount",evidence).changed,true,"clearing OCR is still a correction");
  assert.equal(fieldEvidence(edited,"total_amount",evidence).source,"1070.00");
  assert.equal(data.fields.total_amount,"1070.00","evidence must never mutate the form");
});
test("OCR metadata cannot cross job boundaries or invent a location", () => {
  const info = fieldEvidence(data,"total_amount",{...evidence,jobId:"run-2"});
  assert.deepEqual(info.locations,[]);
  assert.equal(info.raw,undefined);
  assert.equal(info.stale,true);
  assert.equal(info.status,"OCR รอบก่อน");
  const invalid = {...evidence,fields:{total_amount:{...evidence.fields.total_amount,presence:"ambiguous",evidence:[{page:0,line:-1}]}}};
  assert.equal(fieldEvidence(data,"total_amount",invalid).uncertain,true);
  assert.deepEqual(fieldEvidence(data,"total_amount",invalid).locations,[]);
  assert.equal(fieldEvidence({...data,source:undefined},"total_amount",evidence).status,"ข้อมูลในแบบฟอร์ม");
  const lowQuality = {...evidence,fields:{total_amount:{...evidence.fields.total_amount,evidence:[{page:1,line:8,ocr_confidence:.79}]}}};
  assert.equal(fieldEvidence(data,"total_amount",lowQuality).uncertain,true);
  assert.equal(fieldEvidence(data,"total_amount",lowQuality).status,"ควรตรวจต้นฉบับ");
});
test("all seven types expose their active fields without losing hidden values", () => {
  for(const type of expenseDocumentTypes) {
    const form = {...data,type,fields:{...data.fields,posting_amount:"999.00"}};
    const keys = visibleFormFields(form);
    assert.ok(keys.length>0,type);
    assert.ok(!keys.includes("posting_amount"));
    assert.equal(form.fields.posting_amount,"999.00");
    for(const key of keys) assert.ok(formFieldLabel(form,key));
  }
  assert.match(formFieldLabel(data,"items.2.amount"),/แถว 3/);
});
