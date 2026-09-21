import { Card } from "@/components/ui/card";
import { StatusBadge } from "@/components/ui/status-badge";
import { getSnapshot } from "@/lib/api";

export const dynamic = "force-dynamic";

export default async function SystemStatus() {
  const snapshot = await getSnapshot();
  const jobs = snapshot.jobs;
  const metrics = [
    ["รอทำงาน", jobs.queued], ["กำลังทำ", jobs.running], ["ล้มเหลว 24 ชม.", jobs.failed],
    ["Retry", jobs.retries], ["Stale recovery", jobs.stale_reclaims],
    ["คิวเก่าสุด (วินาที)", jobs.oldest_queue_wait_seconds],
    ["Worker heartbeat (วินาที)", jobs.worker_heartbeat_age_seconds], ["Export rows 24 ชม.", jobs.export_rows],
    ["ดาวน์โหลด 24 ชม.", jobs.downloads], ["เวลาทำงานเฉลี่ย (วินาที)", jobs.average_runtime_seconds],
  ] as const;
  return <><p className="eyebrow">Operations</p><h1>สถานะระบบ</h1><p className="intro">สถานะล่าสุดจาก API, PostgreSQL และ worker</p><div className="grid systems">{[["API", snapshot.api], ["ฐานข้อมูล", snapshot.database], ["Worker", snapshot.worker]].map(([label, status]) => <Card className="system-row" key={label}><h2>{label}</h2><StatusBadge status={status} /></Card>)}</div><h2>งานเบื้องหลัง</h2><div className="grid systems">{metrics.map(([label, value]) => <Card className="system-row" key={label}><h3>{label}</h3><strong>{value}</strong></Card>)}</div></>;
}
