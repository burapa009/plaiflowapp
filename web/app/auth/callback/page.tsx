import { Card } from "@/components/ui/card";

export default function AuthCallbackShell() {
  return <><p className="eyebrow">Phase 1 seam</p><h1>ผลการเชื่อมต่อบัญชี</h1><Card className="notice"><strong>ยังไม่เปิดใช้งานการเข้าสู่ระบบ</strong><p>หน้านี้สำรองเส้นทาง callback เท่านั้น และยังไม่รับหรือจัดเก็บ token จากผู้ให้บริการ</p></Card></>;
}
