import config from "./document-forms.json" with { type: "json" };
export type FormAssessment = {ruleVersion:string;action:string;evidenceStatus:string;missing:Record<string,string>;unresolved:Record<string,string>;differences:Record<string,string>;calculated:Record<string,string>;taxStatus:string};
export type CanonicalDocument = { version: number; type: string; fields: Record<string,string>; tables: Record<string,Record<string,string>[]>;assessment?:FormAssessment;source?:{ocr_job_id:string;fields:Record<string,string>;tables:Record<string,Record<string,string>[]>} };
export type FieldRule = {field:string;requirement:"LEGAL_REQUIRED"|"ACCOUNTING_REQUIRED"|"CONDITIONAL_REQUIRED"|"OPTIONAL"|"NOT_APPLICABLE";condition:Record<string,string>;conditionLabel:string;action:string[];reason:string;sourceUrl:string;legalReference:string;effectiveFrom:string;legalEffectiveFrom:string|null;ruleVersion:string;hiddenWhen?:Record<string,string>;evidenceCheck?:boolean;additionalRules?:FieldRule[]};
export type FormType = { label: string; sections: string[]; tables: string[]; roles: string[]; required: string[]; issuance: boolean };
export const formConfig = config as {
 fields: Record<string,{label:string;kind:string;options?:string[][]}>;
 sections: Record<string,{label:string;fields:string[]}>;
 tables: Record<string,{label:string;columns:string[][]}>;
 types: Record<string,FormType>;
 ruleVersion:string;rules:Record<string,Record<string,FieldRule>>;
};
export type FormReference = {id:string;data:CanonicalDocument};
export type DocumentFormResponse = {form:{data:CanonicalDocument;revision:number;ocr_job_id:string;status:string};review_revision:number;legacy_draft_revision?:number;can_edit:boolean;can_confirm:boolean;source_superseded:boolean;references:FormReference[]};

export function decimalCents(value:string): bigint | null {
 if (!/^-?\d{1,12}(\.\d{1,2})?$/.test(value)) return null;
 const negative=value.startsWith("-"), [whole,fraction=""]=value.replace("-","").split(".");
 return (BigInt(whole)*100n+BigInt(fraction.padEnd(2,"0")))*(negative?-1n:1n);
}
export function formatCents(value:bigint):string { const sign=value<0n?"-":""; const n=value<0n?-value:value; return sign+String(n/100n)+"."+String(n%100n).padStart(2,"0"); }
export function sumRows(rows:Record<string,string>[],key:string):string {
 if(!rows.length) return "";
 let sum=0n; for(const row of rows){const n=decimalCents(row[key]??"");if(n===null)return "";sum+=n;} return formatCents(sum);
}

export function conditionState(data:CanonicalDocument, condition:Record<string,string>, row:Record<string,string>={}):"yes"|"no"|"unknown" {
 let unknown=false;
 for(const [key,want] of Object.entries(condition)){
  const value=key.startsWith("$row.")?row[key.slice(5)]:data.fields[key];
  if(want==="@present"){if(!value)return "no";continue;}
  if(!value||value==="unknown"){unknown=true;continue;}
  if(value!==want)return "no";
 }
 return unknown?"unknown":"yes";
}
export function fieldVisible(data:CanonicalDocument,key:string):boolean {
 const rule=formConfig.rules[data.type]?.[key];
 return !!rule&&rule.requirement!=="NOT_APPLICABLE"&&(!rule.hiddenWhen||conditionState(data,rule.hiddenWhen)!=="yes");
}
export function ruleLabel(rule:FieldRule):string {
 return rule.requirement==="LEGAL_REQUIRED"?"จำเป็นตามกฎหมายเมื่อออกเอกสาร / ตรวจภาษี":rule.requirement==="ACCOUNTING_REQUIRED"?"จำเป็นเพื่อบันทึกบัญชี (ข้อกำหนดระบบ)":rule.requirement==="CONDITIONAL_REQUIRED"?`จำเป็นเมื่อ${rule.conditionLabel}`:"ไม่บังคับ";
}
function product(a:string,b:string,percent=false):bigint|null {
 if(!/^(0|[1-9]\d{0,11})(\.\d{1,4})?$/.test(a)||!/^(0|[1-9]\d{0,11})(\.\d{1,4})?$/.test(b))return null;
 const parse=(v:string)=>{const [w,f=""]=v.split(".");return [BigInt(w+f),10n**BigInt(f.length)];};
 const [an,ad]=parse(a),[bn,bd]=parse(b),num=an*bn*(percent?1n:100n),den=ad*bd;
 return (num*2n+den)/(den*2n);
}
export function calculatedAmounts(data:CanonicalDocument):Record<string,string>{
 const out:Record<string,string>={},f=data.fields,typ=formConfig.types[data.type];if(!typ)return out;
 const active=new Set(typ.sections.flatMap(s=>formConfig.sections[s].fields));
 const cents=(key:string)=>decimalCents(f[key]??"");
 const sum=(table:string,key:string):bigint|null=>{const rows=data.tables[table]??[];if(!rows.length)return null;let total=0n;for(const [i,row] of rows.entries()){const n=decimalCents(row[key]||out[`${table}.${i}.${key}`]||"");if(n===null)return null;total+=n;}return total;};
 if(typ.tables.includes("items"))for(const [i,row] of (data.tables.items??[]).entries()){const n=product(row.quantity??"",row.unit_price??""),discount=decimalCents(row.discount??"");if(n!==null&&discount!==null&&n>=discount)out[`items.${i}.amount`]=formatCents(n-discount);}
 if(active.has("price_mode")){
  const items=sum("items","amount"),discount=cents("discount");
  if(items!==null&&discount!==null&&items>=discount){if(f.price_mode==="exclusive")out.subtotal=formatCents(items-discount);if(f.price_mode==="inclusive")out.total_amount=formatCents(items-discount);}
  const base=out.subtotal??f.subtotal??"",vat=product(base,f.vat_rate??"",true);
  if(vat!==null)out.vat_amount=formatCents(vat);
  const v=vat??cents("vat_amount"),b=decimalCents(base);if(b!==null&&v!==null&&!out.total_amount)out.total_amount=formatCents(b+v);
 }
 if(data.type==="billing_note"){const n=sum("references","allocated");if(n!==null)out.total_amount=formatCents(n);}
 if(data.type==="withholding_tax_certificate")for(const [key,dest] of [["amount","income_total"],["tax","withholding_tax"]]){const n=sum("income",key);if(n!==null)out[dest]=formatCents(n);}
 const total=cents("total_amount"),tax=cents("withholding_tax");if(active.has("net_amount")&&total!==null&&tax!==null&&total>=tax)out.net_amount=formatCents(total-tax);
 if(["credit_note","debit_note"].includes(data.type)){const before=cents("before_amount");if(before!==null&&total!==null){const after=before+(data.type==="credit_note"?-total:total);if(after>=0n)out.after_amount=formatCents(after);}}
 return out;
}
export function ruleIssues(data:CanonicalDocument,action="confirm") {
 const missing:Record<string,string>={},unresolved:Record<string,string>={},calculated=calculatedAmounts(data);
 for(const [key,rule] of Object.entries(formConfig.rules[data.type]??{})){
  if(!fieldVisible(data,key))continue;
  for(const r of [rule,...rule.additionalRules??[]]){
   if(action==="confirm"&&r.evidenceCheck===false)continue;
   if((action!=="confirm"&&!r.action.includes(action))||["OPTIONAL","NOT_APPLICABLE"].includes(r.requirement))continue;
   const inspect=(path:string,value:string,row:Record<string,string>={})=>{const state=conditionState(data,r.condition,row);if(state==="unknown")unresolved[path]=`ต้องตรวจสอบเงื่อนไข: ${r.conditionLabel}`;else if(state==="yes"&&!value?.trim()&&!calculated[path])missing[path]=r.reason;};
   const [table,column]=key.split(".");if(!column)inspect(key,data.fields[key]);else {const rows=data.tables[table]??[];if(!rows.length)inspect(key,"");rows.forEach((row,i)=>inspect(`${table}.${i}.${column}`,row[column],row));}
  }
 }
 return {missing,unresolved};
}

// One reading order for the editor and the saved-data print summary.
export function formLayout(data:CanonicalDocument):{kind:"section"|"table";key:string}[] {
 const typ=formConfig.types[data.type]??formConfig.types.unknown;
 const tail=new Set(["amounts","receipt_amounts","withholding","wht","billing","payment","notes","signatures"]);
 const sections=typ.sections.filter(key=>key!=="posting");
 return [
  ...sections.filter(key=>!tail.has(key)).map(key=>({kind:"section" as const,key})),
  ...typ.tables.map(key=>({kind:"table" as const,key})),
  ...sections.filter(key=>tail.has(key)).map(key=>({kind:"section" as const,key})),
 ];
}
