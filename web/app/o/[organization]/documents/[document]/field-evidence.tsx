import { fieldEvidence, type FormEvidence } from "@/lib/document-evidence";
import type { CanonicalDocument } from "@/lib/document-form";

export default function FieldEvidence({data, field, evidence}: {data:CanonicalDocument;field:string;evidence?:FormEvidence}) {
  const info = fieldEvidence(data, field, evidence);
  return <details className="field-evidence" data-changed={info.changed} data-uncertain={info.uncertain}>
    <summary><span>{info.status}</span><span className="field-evidence-action">ดูที่มา</span></summary>
    <div>
      <p><strong>ค่าที่ OCR เติม:</strong> {info.source?.trim() || "ไม่มีค่า OCR ที่บันทึกไว้ · ระบบไม่ได้เดาแทน"}</p>
      {info.stale && <p>ค่านี้มาจาก OCR รอบก่อน จึงไม่แสดงตำแหน่งข้อความจากรอบใหม่ปะปนกัน</p>}
      {info.raw && <p><strong>ข้อความที่อ่านได้:</strong> <q>{info.raw}</q></p>}
      {info.locations.length ? <p>พบที่ {info.locations.map(ref => `หน้า ${ref.page} บรรทัด ${ref.line}`).join(" · ")}</p> : <p>ไม่มีตำแหน่งข้อความสำหรับช่องนี้ โปรดเทียบภาพต้นฉบับ</p>}
      {info.uncertain && <p>ข้อความไม่ชัดเจนหรือพบค่าขัดแย้ง โปรดตรวจต้นฉบับก่อนยืนยัน</p>}
      {info.changed && <p>ค่าในแบบฟอร์มต่างจาก OCR เดิม การแก้ไขไม่เปลี่ยนหลักฐานต้นฉบับ</p>}
    </div>
  </details>;
}
