import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {calculatedAmounts,fieldVisible,ruleIssues,formLayout,type CanonicalDocument} from "../lib/document-form.ts";
type Fixture={name:string;data:CanonicalDocument;missing?:string[];notMissing?:string[];unknown?:string[];hidden?:string[];calculated:Record<string,string>};
const fixtures=JSON.parse(readFileSync(new URL("./fixtures/document-form-rules.json",import.meta.url),"utf8")) as Fixture[];
for(const f of fixtures)test(f.name,()=>{
 const {missing,unresolved}=ruleIssues(f.data);
 for(const k of f.missing??[])assert.ok(missing[k],k);
 for(const k of f.notMissing??[])assert.ok(!missing[k]&&!unresolved[k],k);
 for(const k of f.unknown??[])assert.ok(unresolved[k],k);
 for(const k of f.hidden??[])assert.equal(fieldVisible(f.data,k),false,k);
 assert.deepEqual(calculatedAmounts(f.data),f.calculated);
});
test("document reading order places items before totals without posting controls",()=>{
 const data:CanonicalDocument={version:1,type:"tax_invoice",fields:{posting_amount:"100"},tables:{}};
 const layout=formLayout(data);
 assert.ok(layout.findIndex(s=>s.key==="items")<layout.findIndex(s=>s.key==="amounts"));
 assert.ok(!layout.some(s=>s.key==="posting"));
 assert.equal(data.fields.posting_amount,"100");
});
