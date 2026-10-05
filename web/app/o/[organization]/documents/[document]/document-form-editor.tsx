"use client";

import { useActionState, useEffect, useRef, useState } from "react";
import { useFormStatus } from "react-dom";
import Link from "next/link";
import FormRuleHint from "./form-rule-hint";
import { formLayout, fieldVisible, ruleIssues, calculatedAmounts, decimalCents, formatCents, formConfig, sumRows, type CanonicalDocument, type DocumentFormResponse } from "@/lib/document-form";
import "./document-form.css";

export type DocumentFormState = {error:string;fields:Record<string,string>;saved?:DocumentFormResponse["form"];reviewRevision?:number};
function Actions({canConfirm,disabled}:{canConfirm:boolean;disabled:boolean}) {
 const {pending}=useFormStatus();
 return <div className="card-actions"><button className="secondary-button" name="intent" value="draft" type="submit" formNoValidate disabled={pending||disabled}>{pending?"กำลังบันทึก…":"บันทึกร่าง"}</button>{canConfirm&&<button className="button" name="intent" value="confirm" disabled={pending||disabled}>ยืนยันข้อมูลกับต้นฉบับ</button>}</div>;
}
export default function DocumentFormEditor({initial,action,cancelHref,printHref,readOnly,notice}:{initial:DocumentFormResponse;action:(state:DocumentFormState,form:FormData)=>Promise<DocumentFormState>;cancelHref:string;printHref:string;readOnly:boolean;notice:string}) {
 const [data,setData]=useState<CanonicalDocument>(initial.form.data);
 const [ack,setAck]=useState(false);
 const [state,formAction,pending]=useActionState(async (previous:DocumentFormState,form:FormData)=>{
  const result=await action(previous,form);
  if(result.error)return {...result,saved:previous.saved,reviewRevision:previous.reviewRevision};
  if(result.saved){setData(result.saved.data);setAck(false);}
  return result;
 },{error:"",fields:{}});
 const revision=state.saved?.revision??initial.form.revision;
 const reviewRevision=state.reviewRevision??initial.review_revision;
 const dirty=JSON.stringify(data)!==JSON.stringify(state.saved?.data??initial.form.data);
 const errorRef=useRef<HTMLDivElement>(null);
 const typ=formConfig.types[data.type]??formConfig.types.unknown;
 const rules=formConfig.rules[data.type]??formConfig.rules.unknown;
 const calculated=calculatedAmounts(data),issues=ruleIssues(data);
 useEffect(()=>{if(state.error){document.dispatchEvent(new CustomEvent("review-focus-field"));errorRef.current?.focus();}if(state.saved&&!state.error)document.dispatchEvent(new CustomEvent("review-saved"));},[state]);
 const change=(next:CanonicalDocument)=>{setData(next);setAck(false);document.dispatchEvent(new CustomEvent("review-dirty"));};
 const field=(key:string,label?:string)=>{
  const spec=formConfig.fields[key];if(!spec||!fieldVisible(data,key))return null;
  const id="form-"+key;const required=typ.required.includes(key);const error=state.fields[key];
  const props={id,value:data.fields[key]??"",onChange:(event:React.ChangeEvent<HTMLInputElement|HTMLTextAreaElement|HTMLSelectElement>)=>change({...data,fields:{...data.fields,[key]:event.target.value}}),disabled:readOnly||pending,"aria-required":required,"aria-invalid":!!error,"aria-describedby":id+"-hint"+(error?" "+id+"-error":"")};
  return <div className="field" key={key}><label htmlFor={id}>{label??spec.label}{required?" *":""}</label>{spec.options?<select {...props}><option value="">ยังไม่ระบุ</option>{spec.options.map(([value,label])=><option key={value} value={value}>{label}</option>)}</select>:key==="price_mode"?<select {...props}><option value="">ไม่ระบุ</option><option value="exclusive">ไม่รวม VAT</option><option value="inclusive">รวม VAT</option></select>:spec.kind==="textarea"?<textarea {...props} rows={3} maxLength={4000}/>:<input {...props} type={["date","time"].includes(spec.kind)?spec.kind:"text"} inputMode={["money","signed_money","decimal"].includes(spec.kind)?"decimal":undefined} maxLength={4000}/>} <FormRuleHint data={data} rule={rules[key]} id={id+"-hint"}/>{error&&<p className="form-error" id={id+"-error"}>{error}</p>}{calculated[key]&&<p className="canonical-calculated">ยอดคำนวณ: {calculated[key]} {data.fields.currency} · ไม่ต้องกรอกซ้ำ{data.fields[key]&&decimalCents(data.fields[key])!==decimalCents(calculated[key])?" · ต่างจากยอดต้นฉบับ ต้องตรวจสอบ":""}</p>}{data.source?.fields[key]!==undefined&&data.source.fields[key]!==data.fields[key]&&<p className="canonical-hint">OCR เดิม: {data.source.fields[key]||"ไม่พบค่า"} · ข้อมูลเพิ่มไม่ถือว่ามีบนต้นฉบับ</p>}</div>;
 };
 function updateRow(table:string,index:number,key:string,value:string){
  const rows=(data.tables[table]??[]).map(row=>({...row}));
  rows[index][key]=value;
  if(key==="document_id"&&table==="references"){
   const ref=initial.references.find(r=>r.id===value);
   for(const field of ["document_number","issue_date","total_amount","paid_amount"])rows[index][field]=ref?.data.fields[field]??"";
  }
  change({...data,tables:{...data.tables,[table]:rows}});
 }
 const renderSection=(section:string)=>{
  const spec=formConfig.sections[section];
  const visible=spec.fields.filter(key=>fieldVisible(data,key));
  if(!visible.length)return null;
  const primary=visible.filter(key=>rules[key].requirement!=="OPTIONAL"||!!formConfig.fields[key].options);
  const optional=visible.filter(key=>!primary.includes(key));
  const render=(key:string)=>field(key,formConfig.fields[key].label.replace("ฝ่ายที่หนึ่ง",typ.roles[0]).replace("ฝ่ายที่สอง",typ.roles[1]));
  return <section className="ocr-review-card" key={section}><h3>{spec.label}</h3>{primary.length>0&&<div className="canonical-fields">{primary.map(render)}</div>}{optional.length>0&&<details open={optional.some(k=>!!state.fields[k])||undefined}><summary>ข้อมูลเพิ่มเติม</summary><div className="canonical-fields">{optional.map(render)}</div></details>}</section>;
 };
 const renderTable=(table:string)=>{
    const spec=formConfig.tables[table],rows=data.tables[table]??[];
    return <section className="ocr-review-card" key={table} id={"table-"+table} tabIndex={-1}><h3>{spec.label}</h3>
     {table==="references"&&<p>เลือกเอกสารที่ยืนยันแล้วในทีม (ล่าสุด 100 รายการ) หรือกรอกเลขที่และหลักฐานต้นทางภายนอก</p>}
     {state.fields[table]&&<p className="form-error">{state.fields[table]}</p>}
     {rows.map((row,index)=><fieldset className="canonical-row" key={index} disabled={readOnly||pending}><legend>รายการ {index+1}</legend><div className="canonical-fields">{spec.columns.filter(([key])=>fieldVisible(data,`${table}.${key}`)).map(([key,label,kind])=>{
      const path=`${table}.${index}.${key}`,id=`form-${path}`,error=state.fields[path];
      const accessibility={"aria-invalid":!!error,"aria-describedby":id+"-hint"+(error?" "+id+"-error":"")};
      return <div className="field" key={key}><label htmlFor={id}>{label}</label>{kind==="reference"?<select {...accessibility} id={id} value={row[key]??""} onChange={e=>updateRow(table,index,key,e.target.value)}><option value="">ยังไม่เลือก</option>{initial.references.filter(r=>data.type!=="billing_note"||r.data.type==="invoice").map(ref=><option key={ref.id} value={ref.id}>{ref.data.fields.document_number||ref.id} · {formConfig.types[ref.data.type]?.label}</option>)}</select>:<input {...accessibility} id={id} value={row[key]??""} type={kind==="date"?"date":"text"} inputMode={["money","decimal","signed_money"].includes(kind)?"decimal":undefined} readOnly={table==="references"&&!!row.document_id&&["document_number","issue_date","total_amount","paid_amount"].includes(key)} onChange={e=>updateRow(table,index,key,e.target.value)} maxLength={2000}/>}
       <FormRuleHint data={data} row={row} rule={rules[`${table}.${key}`]} id={id+"-hint"}/>{error&&<p className="form-error" id={id+"-error"}>{error}</p>}{calculated[path]&&<p>ยอดคำนวณ {calculated[path]} · ไม่ต้องกรอกซ้ำ</p>}</div>;
     })}</div>{table==="transactions"&&<p>การจับคู่: {row.document_id&&row.allocated?"ระบุเอกสารและยอดแล้ว (ยังไม่ลงบัญชี)":"ยังไม่จับคู่"}</p>}{table==="references"&&decimalCents(row.total_amount??"")!==null&&decimalCents(row.paid_amount??"")!==null&&<p>คงเหลือตามข้อมูลที่ทราบ: {formatCents(decimalCents(row.total_amount)!-decimalCents(row.paid_amount)!)}</p>}<button type="button" className="secondary-button" disabled={index===0} onClick={()=>{const next=[...rows];[next[index-1],next[index]]=[next[index],next[index-1]];change({...data,tables:{...data.tables,[table]:next}});}}>เลื่อนแถว {index+1} ขึ้น</button><button type="button" className="secondary-button" onClick={()=>change({...data,tables:{...data.tables,[table]:rows.filter((_,i)=>i!==index)}})}>นำแถว {index+1} ออก</button></fieldset>)}
     <button type="button" className="secondary-button" disabled={readOnly||pending||rows.length>=200} onClick={()=>change({...data,tables:{...data.tables,[table]:[...rows,{}]}})}>เพิ่มรายการ{spec.label}</button>
     {table==="references"&&data.type==="billing_note"&&<p>ยอดรวมที่นำมาวางบิล: {sumRows(rows,"allocated")||"—"} {data.fields.currency}</p>}
     {table==="income"&&<p>เงินได้รวม {sumRows(rows,"amount")||"—"} · ภาษีหักรวม {sumRows(rows,"tax")||"—"}</p>}
    </section>;
   };
 return <section className="ocr-review-panel document-form-panel">
  <h2>ข้อมูลเอกสารบัญชี</h2>
  <p>ตรวจเทียบต้นฉบับก่อนยืนยัน · ข้อมูลไม่ครบยังส่งตรวจได้ · ไม่ใช่การรับรองสิทธิภาษี</p>
  {initial.source_superseded&&<p className="form-error" role="alert">OCR เปลี่ยนหลังการบันทึก กรุณาตรวจทุกส่วนใหม่</p>}
  {notice&&<p role="status">{notice==="confirmed"?"ยืนยันข้อมูลแล้ว":notice==="draft-saved"?"บันทึกร่างแล้ว":""}</p>}
  <form action={formAction} className="ocr-review-form work-form" onReset={e=>e.preventDefault()}>
   <input type="hidden" name="document_data" value={JSON.stringify({version:data.version,type:data.type,fields:data.fields,tables:data.tables})}/><input type="hidden" name="revision" value={revision}/><input type="hidden" name="review_revision" value={reviewRevision}/>
   {state.error&&<div ref={errorRef} role="alert" tabIndex={-1} className="ocr-review-alert"><p>{state.error}</p><ul>{Object.entries(state.fields).map(([key,message])=><li key={key}><button type="button" className="ocr-review-error-link" onClick={()=>{const el=document.getElementById("form-"+key)??document.getElementById("table-"+key.split(".")[0]);let parent=el?.parentElement;while(parent){if(parent instanceof HTMLDetailsElement)parent.open=true;parent=parent.parentElement;}el?.focus();el?.scrollIntoView({block:"center"});}}>{formConfig.fields[key]?.label??formConfig.tables[key]?.label??key}: {message}</button></li>)}</ul></div>}
   {revision>0&&!dirty&&!state.error&&<p role="status">{(state.saved??initial.form).status==="Confirmed"?"ยืนยันเอกสารแล้ว":"บันทึกร่างแล้ว"} · ฉบับ {revision}</p>}
   <div className="field"><label htmlFor="form-type">ประเภทเอกสาร</label><select id="form-type" value={data.type} disabled={readOnly||pending} onChange={e=>change({...data,type:e.target.value})}>{Object.entries(formConfig.types).map(([key,type])=><option value={key} key={key}>{type.label}</option>)}</select></div>
   {data.type==="bank_slip"&&<p className="notice">ภาพ Slip เป็นหลักฐานประกอบ ยังไม่ใช่ผลตรวจสอบกับธนาคาร</p>}
   {["credit_note","debit_note"].includes(data.type)&&<p className="notice">กรอกยอดปรับเป็นค่าบวก · {data.type==="credit_note"?"ลด":"เพิ่ม"}มูลค่าตามประเภท ไม่แก้ทับเอกสารต้นทาง</p>}
   {(Object.keys(issues.missing).length>0||Object.keys(issues.unresolved).length>0)&&<details className="canonical-evidence"><summary>หลักฐานไม่ครบ / เงื่อนไขต้องตรวจสอบ ({Object.keys(issues.missing).length+Object.keys(issues.unresolved).length})</summary><p>ยังบันทึกและยืนยันข้อมูลกับต้นฉบับได้ โดยไม่รับรองสิทธิภาษี</p><ul>{Object.entries({...issues.missing,...issues.unresolved}).map(([key,message])=><li key={key}>{formConfig.fields[key]?.label??key}: {message}</li>)}</ul></details>}
   {formLayout(data).map(({kind,key})=>kind==="section"?renderSection(key):renderTable(key))}
   {data.assessment&&<details><summary>ผลตรวจที่บันทึก · รุ่น {data.assessment.ruleVersion}</summary><p>{data.assessment.evidenceStatus==="incomplete"?"หลักฐานไม่ครบ / ต้องตรวจสอบ":data.assessment.action==="draft"?"ฉบับร่าง · ยังไม่ประเมินความครบถ้วน":"ตรวจข้อมูลแล้ว"} · ยังไม่ประเมินสิทธิภาษี{dirty?" · มีการแก้ไขที่ยังไม่บันทึก":""}</p>{Object.entries(data.assessment.differences).map(([key,value])=><p key={key}>{formConfig.fields[key]?.label??key}: {value}</p>)}</details>}
   <div className="ocr-review-card"><p>เอกสารนำเข้าเก็บต้นฉบับไว้ · ยังไม่รองรับออกเอกสารทางการประเภทนี้จากฟอร์มนี้</p>{printHref&&revision>0&&<a className="secondary-button" href={printHref} target="_blank" rel="noopener noreferrer">พิมพ์สรุปข้อมูลที่บันทึก</a>}</div>
   <div className="canonical-actions">{initial.can_confirm&&<label><input id="form-review_ack" name="review_ack" type="checkbox" value="1" checked={ack} onChange={e=>setAck(e.target.checked)} disabled={readOnly||pending} aria-invalid={!!state.fields.review_ack} aria-describedby={state.fields.review_ack?"form-review_ack-error":undefined}/> ฉันตรวจข้อมูลทั้งเอกสารเทียบกับต้นฉบับแล้ว</label>}{state.fields.review_ack&&<p className="form-error" id="form-review_ack-error">{state.fields.review_ack}</p>}<Actions canConfirm={initial.can_confirm} disabled={readOnly}/><Link href={cancelHref} className="secondary-button">กลับรายการ</Link></div>
  </form>
 </section>;
}
