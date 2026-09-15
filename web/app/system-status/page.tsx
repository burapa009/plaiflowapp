import { Card } from "@/components/ui/card";
import { StatusBadge } from "@/components/ui/status-badge";
import { getSnapshot } from "@/lib/api";

export const dynamic = "force-dynamic";

export default async function SystemStatus() {
  const snapshot = await getSnapshot();
  return <><p className="eyebrow">Operations</p><h1>สถานะระบบ</h1><p className="intro">สถานะล่าสุดจาก API, PostgreSQL และ heartbeat ของ worker</p><div className="grid systems">{[['API', snapshot.api], ['ฐานข้อมูล', snapshot.database], ['Worker', snapshot.worker]].map(([label, status]) => <Card className="system-row" key={label}><h2>{label}</h2><StatusBadge status={status} /></Card>)}</div></>;
}
