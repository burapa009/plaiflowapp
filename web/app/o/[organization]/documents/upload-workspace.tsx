import Link from "next/link";
import { UploadForm } from "./upload-form";
import { DriveImportForm } from "./drive-import-form";
import { DocumentIcon } from "./document-visuals";

export function UploadWorkspace({ organization, canManage, preview = false }: { organization: string; canManage: boolean; preview?: boolean }) {
  return <section id="document-upload" className="inbox-upload-workspace" aria-labelledby="upload-heading">
    <div><h2 id="upload-heading">อัปโหลดเอกสารบัญชี</h2><p className="inbox-description">อัปโหลดเอกสาร แล้วตรวจข้อมูลเทียบกับต้นฉบับก่อนยืนยัน</p><UploadForm organization={organization} preview={preview} /></div>
    <aside className="inbox-workflow"><h3>ขั้นตอนการทำงาน</h3><ol><li>อัปโหลดเอกสาร</li><li>อ่านข้อมูลจากเอกสาร</li><li>จัดหมวดเมื่อเปิดใช้</li><li>ตรวจสอบและแก้ไข</li><li>ยืนยันข้อมูลที่ตรวจแล้ว</li></ol></aside>
    <div className="inbox-sources"><span className="document-icon-circle"><DocumentIcon kind="layers" /></span><h3>เอกสารเป็นระเบียบ<br />ตรวจสอบได้ทุกขั้นตอน</h3><p>เก็บต้นฉบับไว้เทียบกับข้อมูลที่อ่านได้</p>{canManage && !preview && <><details><summary>นำเข้าจาก Google Drive</summary><DriveImportForm organization={organization} /></details><Link href={`/o/${encodeURIComponent(organization)}/connections`}>จัดการการเชื่อมต่อ</Link></>}</div>
  </section>;
}
