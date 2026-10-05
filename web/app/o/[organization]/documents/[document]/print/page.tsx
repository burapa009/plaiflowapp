import { sessionGET } from "@/lib/session-api";
import { formLayout, fieldVisible, formConfig, type DocumentFormResponse } from "@/lib/document-form";
import PrintButton from "./print-button";
import "./print.css";

function displayValue(value:string|undefined,kind:string,options?:string[][]):string {
 if(!value)return "—";
 const label=options?.find(([v])=>v===value)?.[1];if(label)return label;
 if(kind==="date"&&/^\d{4}-\d{2}-\d{2}$/.test(value)){
  const date=new Date(value+"T00:00:00Z");if(!Number.isNaN(date.getTime()))return new Intl.DateTimeFormat("th-TH",{dateStyle:"medium",timeZone:"UTC"}).format(date);
 }
 if(["money","signed_money"].includes(kind)&&/^-?\d+(\.\d{1,2})?$/.test(value)){
  const [whole,fraction=""]=value.split(".");return whole.replace(/\B(?=(\d{3})+(?!\d))/g,",")+"."+fraction.padEnd(2,"0");
 }
 return value;
}

export default async function DocumentPrint({params}:{params:Promise<{organization:string;document:string}>}){
 const {organization,document}=await params;
 const response=await sessionGET(`/v1/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}/form`);
 if(!response?.ok)return <p role="alert">ยังเปิดข้อมูลที่บันทึกไม่ได้ หรือคุณไม่มีสิทธิ์</p>;
 const {form,source_superseded}=await response.json() as DocumentFormResponse;
 if(!form.revision)return <p>บันทึกข้อมูลก่อนพิมพ์สรุป</p>;
 const data=form.data,typ=formConfig.types[data.type];
 if(!typ)return <p role="alert">ไม่รองรับประเภทเอกสาร</p>;
 const renderSection=(section:string)=>!formConfig.sections[section].fields.some(key=>fieldVisible(data,key)&&data.fields[key])?null:<section key={section}><h2>{formConfig.sections[section].label}</h2><dl>{formConfig.sections[section].fields.filter(key=>fieldVisible(data,key)&&data.fields[key]).map(key=><div key={key}><dt>{formConfig.fields[key].label.replace("ฝ่ายที่หนึ่ง",typ.roles[0]).replace("ฝ่ายที่สอง",typ.roles[1])}</dt><dd>{displayValue(data.fields[key],formConfig.fields[key].kind,formConfig.fields[key].options)}</dd></div>)}</dl></section>;
 const renderTable=(table:string)=>!data.tables[table]?.length?null:<section key={table}><h2>{formConfig.tables[table].label}</h2><table><thead><tr>{formConfig.tables[table].columns.filter(([key])=>fieldVisible(data,`${table}.${key}`)).map(([key,label])=><th key={key}>{label}</th>)}</tr></thead><tbody>{data.tables[table].map((row,index)=><tr key={index}>{formConfig.tables[table].columns.filter(([key])=>fieldVisible(data,`${table}.${key}`)).map(([key,,kind])=><td key={key} className={["money","signed_money","decimal"].includes(kind)?"print-number":undefined}>{displayValue(row[key],kind)}</td>)}</tr>)}</tbody></table></section>;
 return <article className="document-print"><PrintButton/><h1>สรุปข้อมูลจากระบบ — {typ.label}</h1>
  <p>ไม่ใช่ต้นฉบับที่ออกโดยคู่ค้า และไม่ใช่เอกสารทางการที่ออกใหม่</p>
  <p>{form.status==="Draft"?"ฉบับร่าง":source_superseded?"ข้อมูลที่ยืนยันจาก OCR รุ่นก่อน":"ข้อมูลที่ยืนยันแล้ว"} · ฉบับ {form.revision}</p>
  {source_superseded&&<p role="alert">มีผล OCR รุ่นใหม่แล้ว สรุปนี้ใช้ข้อมูลที่บันทึกจากรุ่นก่อน ต้องตรวจสอบและยืนยันใหม่ก่อนใช้งาน</p>}
  <p>{data.assessment?.evidenceStatus==="incomplete"?"หลักฐานไม่ครบ / ต้องตรวจสอบ":data.assessment?.action==="draft"?"ยังไม่ประเมินความครบถ้วน":"สถานะตามการตรวจข้อมูล"} · ยังไม่ประเมินสิทธิภาษี · กฎที่บันทึก {data.assessment?.ruleVersion??"ยังไม่มีผลตรวจ"}</p>
  {formLayout(data).map(({kind,key})=>kind==="section"?renderSection(key):renderTable(key))}
  {!!data.assessment&&<section><h2>ผลตรวจข้อมูลที่บันทึก</h2><dl>{Object.entries(data.assessment.calculated).map(([key,value])=><div key={key}><dt>ยอดคำนวณ · {formConfig.fields[key]?.label??key}</dt><dd>{displayValue(value,"money")} {data.fields.currency}</dd></div>)}</dl>{Object.entries({...data.assessment.missing,...data.assessment.unresolved,...data.assessment.differences}).map(([key,message])=><p key={key}>{(formConfig.fields[key]?.label??key).replace("ฝ่ายที่หนึ่ง",typ.roles[0]).replace("ฝ่ายที่สอง",typ.roles[1])}: {message}</p>)}</section>}
 </article>;
}
