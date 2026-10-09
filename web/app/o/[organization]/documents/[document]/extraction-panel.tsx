import { sessionGET, sessionPOST } from "@/lib/session-api";
import { revalidatePath } from "next/cache";
import type { AccountingResult } from "./accounting-result";
import DocumentFormEditor, { type DocumentFormState } from "./document-form-editor";
import { formConfig, type CanonicalDocument, type DocumentFormResponse } from "@/lib/document-form";
export { documentTypeLabels } from "@/lib/document-types";
type Field = { presence: string; raw: string; normalized: string; confidence: string; evidence: { page: number; line: number }[] };
type Warning = { code: string; field: string; severity: string };
export type Extraction = {
  draft: { document_type: string; fields: Record<string, Field>; warnings: Warning[]; accounting?: AccountingResult; classification?: { document_type: string; effective_type: string; confidence: number; requires_review: boolean; candidate_types: { type: string; confidence: number }[]; signals: string[] } };
  ocr_job_id: string;
  revision: number;
  source_superseded: boolean;
  possible_duplicate?: { possible_duplicate: boolean; matched_document_id: string; duplicate_confidence: number } | null;
  confirmed: { values: Record<string, string>; decisions?: Record<string, string>; confirmed_at: string } | null;
  review_enabled?: boolean;
  saved_review?: { revision: number; values: Record<string, string>; decisions: Record<string, string> } | null;
	returned_review?: { reason_code: string; private_note: string; returned_at: string } | null;
	review_history?: { revision: number; values: Record<string, string>; decisions: Record<string, string>; updated_by: string; updated_at: string }[];
};
export default async function ExtractionPanel({organization,document,data,role,cancelHref,preview=false,result=""}: {
 organization:string;document:string;data:Extraction;role:string;cancelHref:string;preview?:boolean;previewURL?:string;nextHref?:string;organizationName?:string;branchType?:string;result?:string;attachmentCount?:number;
}) {
 const path=`/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}`;
 let response:DocumentFormResponse;
 if(preview){
  const canonical:CanonicalDocument={version:1,type:formConfig.types[data.draft.document_type]?data.draft.document_type:"unknown",fields:{},tables:{}};
  for(const [key,f] of Object.entries(data.draft.fields)) canonical.fields[key]=data.confirmed?.values[key]??f.normalized??"";
  canonical.tables.items=(data.draft.accounting?.items??[]).map(item=>Object.fromEntries(Object.entries(item).map(([k,v])=>[k,v==null?"":String(v)])));
  response={form:{data:canonical,revision:0,ocr_job_id:data.ocr_job_id,status:"Draft"},review_revision:data.revision,can_edit:true,can_confirm:true,source_superseded:false,references:[]};
 }else{
  const fetched=await sessionGET(`/v1${path}/form`);
  if(!fetched?.ok) return <section className="ocr-review-card" role="alert"><h2>ยังเปิดฟอร์มไม่ได้</h2><p>ข้อมูลฟอร์มยังไม่พร้อม กรุณาลองใหม่ หรือติดต่อผู้ดูแลระบบ</p></section>;
  response=await fetched.json() as DocumentFormResponse;
 }
 async function save(_state:DocumentFormState,form:FormData):Promise<DocumentFormState>{
  "use server";
  if(preview)return {error:"หน้าตัวอย่างไม่บันทึกข้อมูลจริง",fields:{}};
  const raw=String(form.get("document_data")??"");
  if(raw.length>256*1024)return {error:"ข้อมูลมากเกินขอบเขต",fields:{}};
  let canonical:CanonicalDocument;
  try{canonical=JSON.parse(raw) as CanonicalDocument;}catch{return {error:"ข้อมูลฟอร์มไม่ถูกต้อง",fields:{}};}
  const confirm=form.get("intent")==="confirm";
  if(confirm&&(form.get("review_ack")!=="1"||form.get("amount_ack")!=="1"))return {error:"กรุณาตรวจยอดเงินและข้อมูลทั้งเอกสารก่อนยืนยัน",fields:{review_ack:"ต้องตรวจและยืนยันทั้งสองรายการ"}};
  const saved=await sessionPOST(`/v1${path}/form`,JSON.stringify({data:canonical,ocr_job_id:response.form.ocr_job_id,expected_revision:Number(form.get("revision")),expected_review_revision:Number(form.get("review_revision")),expected_legacy_draft_revision:response.legacy_draft_revision??0,confirm,acknowledged:form.get("review_ack")==="1"}));
  if(!saved?.ok){
   const detail=await saved?.json().catch(()=>({})) as {fields?:Record<string,string>}|undefined;
   return {error:saved?.status===409?"ข้อมูลเปลี่ยนแล้ว กรุณาโหลดใหม่ก่อนบันทึก":saved?.status===422?"กรุณาตรวจข้อมูลที่ระบุ":"บันทึกไม่สำเร็จ กรุณาลองใหม่",fields:detail?.fields??{}};
  }
  const record=await saved.json() as DocumentFormResponse["form"];
  revalidatePath(path);revalidatePath(`/o/${encodeURIComponent(organization)}/documents`);
  return {error:"",fields:{},saved:record,reviewRevision:confirm?Number(form.get("review_revision"))+1:Number(form.get("review_revision"))};
 }
 return <DocumentFormEditor key={data.ocr_job_id} initial={response} action={save} cancelHref={cancelHref} printHref={preview?"":`${path}/print`} readOnly={!response.can_edit||!["Owner","Admin","Member"].includes(role)} notice={result} />;
}
