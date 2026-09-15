import { Card } from "@/components/ui/card";
import { StatusBadge } from "@/components/ui/status-badge";
import { getSnapshot } from "@/lib/api";

export const dynamic = "force-dynamic";

export default async function Overview() {
  const snapshot = await getSnapshot();
  const metrics = [
    ["รับเข้ามา", snapshot.counts.received], ["ประมวลผลแล้ว", snapshot.counts.processed],
    ["ข้ามอย่างปลอดภัย", snapshot.counts.ignored], ["รอลองใหม่", snapshot.counts.retryable], ["ต้องแก้ไข", snapshot.counts.failed],
  ] as const;
  return <>
    <p className="eyebrow">24 ชั่วโมงล่าสุด</p>
    <h1>ภาพรวม Inbound Event</h1>
    <p className="intro">ดูจำนวนงานและสุขภาพของ pipeline จากข้อมูลจริง เพื่อรู้ทันทีว่าระบบว่างหรือมีส่วนใดต้องดูแล</p>
    <div className="grid metrics" aria-label="จำนวน Inbound Event">
      {metrics.map(([label, value]) => <Card key={label}><p className="metric-label">{label}</p><p className="metric-value">{value.toLocaleString("th-TH")}</p></Card>)}
    </div>
    <h2 className="section-heading">สุขภาพระบบ</h2>
    <div className="grid systems">
      {[['API', snapshot.api], ['ฐานข้อมูล', snapshot.database], ['Worker', snapshot.worker]].map(([label, status]) => <Card className="system-row" key={label}><h2>{label}</h2><StatusBadge status={status} /></Card>)}
    </div>
    {Object.values(snapshot.counts).every((count) => count === 0) && <Card className="notice"><strong>ยังไม่มี Inbound Event</strong><p>ระบบพร้อมรับ webhook แรก เมื่อเชื่อม LINE staging แล้วข้อมูลจะปรากฏที่นี่</p></Card>}
  </>;
}
