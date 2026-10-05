import { conditionState, ruleLabel, type CanonicalDocument, type FieldRule } from "@/lib/document-form";

export default function FormRuleHint({rule,id,data,row}:{rule:FieldRule;id:string;data:CanonicalDocument;row?:Record<string,string>}) {
 const rules=[rule,...rule.additionalRules??[]];
 const active=rules.find(r=>conditionState(data,r.condition,row)==="yes")??rules.find(r=>conditionState(data,r.condition,row)==="unknown");
 return <div id={id} className="canonical-hint"><span>{active?ruleLabel(active):"ไม่บังคับในกรณีที่ระบุ"}</span>{active&&conditionState(data,active.condition,row)==="unknown"&&<span> · ต้องตรวจสอบเงื่อนไข</span>}<details><summary>เหตุผลและกฎที่ใช้</summary>{rules.map((r,i)=><div key={i}><p>{ruleLabel(r)} · {r.reason}</p><p>{r.legalReference||"นโยบายระบบ"} · รุ่น {r.ruleVersion}</p>{r.sourceUrl!=="system_policy"&&<a href={r.sourceUrl} target="_blank" rel="noopener noreferrer">ที่มา</a>}</div>)}</details></div>;
}
