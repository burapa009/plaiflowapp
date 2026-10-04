import ReviewPreview from "../review-preview/page";
import { DocumentTable } from "../o/[organization]/documents/document-table";
import { notFound } from "next/navigation";
import { DocumentIcon } from "../o/[organization]/documents/document-visuals";
import { UploadWorkspace } from "../o/[organization]/documents/upload-workspace";
import "../o/[organization]/documents/inbox.css";

export default async function LocalPreview({ searchParams }: { searchParams: Promise<{ selected?: string }> }) {
  const { selected } = await searchParams;
  if (process.env.NODE_ENV !== "development") notFound();
  return <section className="document-preview" aria-label="ตัวอย่างหน้าตาเอกสารด้วยข้อมูลจำลอง">
    <p className="notice" role="status">หน้าตัวอย่างในเครื่อง · ข้อมูลจำลอง ไม่บันทึกข้อมูลจริง</p>
    <UploadWorkspace organization="preview" canManage={false} preview />
    <div className="documents-search-row"><form className="documents-search" method="get"><DocumentIcon kind="search" /><input aria-label="ค้นหาเอกสาร" placeholder="ค้นหาชื่อไฟล์เอกสาร..." /><kbd>Ctrl K</kbd></form><a className="documents-upload-action" href="#document-upload"><DocumentIcon kind="upload" />อัปโหลดเอกสาร</a></div>
    <header className="documents-toolbar"><div className="documents-title"><span className="document-icon-circle"><DocumentIcon kind="file" /></span><div><h1>เอกสารของทีม</h1><p>จัดเก็บ ค้นหา และจัดการเอกสารของทีมง่ายๆ ในที่เดียว</p></div></div><nav className="documents-toolbar-actions" aria-label="เครื่องมือเอกสาร"><span className="documents-demo-action"><DocumentIcon kind="layers" />มุมมอง</span><span className="documents-demo-action"><DocumentIcon kind="settings" />ตั้งค่า</span><span className="documents-demo-action"><DocumentIcon kind="download" />ส่งออก</span></nav></header>
    <nav className="documents-tabs" aria-label="สถานะเอกสาร"><span className="documents-demo-tab active">ทั้งหมด</span><span className="documents-demo-tab">พร้อมใช้</span><span className="documents-demo-tab">เก็บถาวร</span><span className="documents-demo-tab">ถังขยะ</span><span className="documents-demo-tab">กำลังตรวจ</span><span>รายการหน้านี้ <b>2</b> รายการ</span></nav>
    <div className="documents-filters"><label>ช่วงวันที่<div className="documents-date-range"><DocumentIcon kind="calendar" /><input type="date" aria-label="ตั้งแต่วันที่" /><span>–</span><input type="date" aria-label="ถึงวันที่" /></div></label><label>ช่องทาง<select aria-label="ช่องทาง"><option>ทั้งหมด</option></select></label><label>สถานะ<select aria-label="สถานะ"><option>ทั้งหมด</option></select></label><span className="documents-reset">รีเซ็ตตัวกรอง</span><span className="documents-demo-action search"><DocumentIcon kind="search" />ค้นหา</span></div>
    <section className="inbox-results"><DocumentTable organization="preview" canManage={false} documentHref={() => "/local-preview?selected=preview#inbox-review"} changeStatus={async () => { "use server"; }} documents={[
      { id: "preview", filename: "INV-20261003-001.pdf", mime: "application/pdf", size: 1258291, status: "Available", accepted_at: "2026-10-03T00:00:00Z", source_channel: "Web" },
      { id: "preview-2", filename: "receipt.png", mime: "image/png", size: 345600, status: "Checking", accepted_at: "2026-10-02T00:00:00Z", source_channel: "LINE" }
    ]} /></section>
    {selected === "preview" && <section id="inbox-review" className="inbox-review" aria-label="ตรวจเอกสารจำลอง"><ReviewPreview /></section>}
  </section>;
}
