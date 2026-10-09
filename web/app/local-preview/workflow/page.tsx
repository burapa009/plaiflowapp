import { notFound } from "next/navigation";
import WorkflowInbox from "../../o/[organization]/documents/workflow-inbox";
import { UploadWorkspace } from "../../o/[organization]/documents/upload-workspace";
import "../../o/[organization]/documents/inbox.css";

export default function Page() {
  if (process.env.NODE_ENV !== "development") notFound();
  return <><p role="status">ข้อมูลจำลอง · ตรวจหน้ารายการที่ใช้ component เดียวกับระบบจริง</p><WorkflowInbox documents={[
    {id:"one",filename:"ตัวอย่างใบแจ้งหนี้.pdf",mime:"application/pdf",size:8192,status:"Available",accepted_at:"2026-10-09T08:00:00Z",source_channel:"Web",stage:"waiting",typeLabel:"ยังไม่มีข้อมูล",ocrDisabled:false},
    {id:"two",filename:"ใบเสร็จสำนักงาน.jpg",mime:"image/jpeg",size:65536,status:"Available",accepted_at:"2026-10-09T08:00:00Z",source_channel:"LINE",stage:"review",typeLabel:"ใบเสร็จรับเงิน",ocrDisabled:false},
    {id:"three",filename:"ใบกำกับภาษี.pdf",mime:"application/pdf",size:8192,status:"Available",accepted_at:"2026-10-09T08:00:00Z",source_channel:"Drive",stage:"confirmed",typeLabel:"ใบกำกับภาษีเต็มรูป",ocrDisabled:false},
    {id:"four",filename:"ใบเสร็จที่อ่านไม่สำเร็จ.png",mime:"image/png",size:8192,status:"Available",accepted_at:"2026-10-09T08:00:00Z",stage:"error",typeLabel:"ยังไม่มีข้อมูล",ocrDisabled:false},
  ]} organization="preview" canManage={false} changeStatus={async()=>{"use server";throw new Error("Preview only");}} upload={<UploadWorkspace organization="preview" canManage={false} preview />} tools={null} search={<form className="workflow-search"><div><input aria-label="ค้นหาชื่อไฟล์เอกสาร" placeholder="ค้นหาเอกสาร…" disabled /></div></form>} notices={null} pagination={null}/></>;
}
