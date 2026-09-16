import Link from "next/link";
import { getAPIBaseURL } from "@/lib/api-config";
import { formatSatang, intervalLabel, isBillingInterval, type BillingInterval, type PlanDefinition } from "@/lib/pricing";

const intervals: BillingInterval[] = ["monthly", "six_months", "yearly"];
const featureLabels: Record<string, string> = {
  "tasks.create": "สร้างและติดตามงาน",
  "line.deliver": "แจ้งเตือนผ่าน LINE",
  "assistant.use": "ผู้ช่วย PlaiFlow",
  "business_contacts.manage": "จัดการคู่ค้า",
  "business_contacts.import": "นำเข้าคู่ค้าจาก CSV/XLSX",
  "business_contacts.export.csv": "ส่งออกคู่ค้าเป็น CSV",
  "business_contacts.export.xlsx": "ส่งออกคู่ค้าเป็น XLSX",
  "business_contacts.export.drive": "เชื่อมต่อ Google Drive",
};

async function loadCatalog() {
  try {
    const response = await fetch(`${getAPIBaseURL(process.env)}/v1/plans`, { cache: "no-store", signal: AbortSignal.timeout(5000) });
    if (!response.ok) return null;
    return await response.json() as { plans: PlanDefinition[]; billing_enabled: boolean };
  } catch {
    return null;
  }
}

export default async function PricingPage({ searchParams }: { searchParams: Promise<{ interval?: string }> }) {
  const query = await searchParams;
  const interval = isBillingInterval(query.interval) ? query.interval : "monthly";
  const catalog = await loadCatalog();

  return <section className="pricing-page">
    <div className="work-header"><div><p className="eyebrow">Early Access</p><h1>แพ็กเกจ PlaiFlow</h1><p className="intro">เปรียบเทียบสิทธิ์และยอดชำระล่วงหน้าจากข้อมูลเดียวกับที่เซิร์ฟเวอร์ใช้อนุญาตฟีเจอร์</p></div><Link className="secondary-button" href="/">กลับหน้าหลัก</Link></div>
    <nav aria-label="เลือกรอบแพ็กเกจ" className="interval-tabs">{intervals.map((item) => <Link key={item} aria-current={item === interval ? "page" : undefined} href={`/pricing?interval=${item}`}>{intervalLabel(item)}</Link>)}</nav>
    {!catalog ? <div className="notice" role="alert"><h2>ยังโหลดแพ็กเกจไม่ได้</h2><p>กรุณาลองใหม่อีกครั้ง</p></div> : <>
      <div className="pricing-grid">{catalog.plans.map((plan) => <PlanCard key={plan.key} plan={plan} interval={interval} />)}</div>
      <p className="pricing-note">ราคาที่แสดงเป็นยอดตามรอบแพ็กเกจ ยังไม่เปิดรับชำระเงินในช่วง Early Access</p>
    </>}
  </section>;
}

function PlanCard({ plan, interval }: { plan: PlanDefinition; interval: BillingInterval }) {
  const price = plan.prices[interval];
  const features = Object.entries(plan.entitlements).filter(([key, enabled]) => enabled && featureLabels[key]).map(([key]) => featureLabels[key]);
  return <article className={`plan-card${plan.recommended ? " recommended" : ""}`}>
    {plan.recommended && <span className="status-pill">แนะนำ</span>}
    <h2>{plan.name}</h2>
    <p className="plan-price"><strong>{formatSatang(price.total_satang)}</strong><span> / {intervalLabel(interval)}</span></p>
    {price.months > 1 && <p className="field-help">เฉลี่ย {formatSatang(price.effective_month_satang)} / เดือน{price.saving_percent > 0 ? ` · ประหยัด ${price.saving_percent}%` : ""}</p>}
    <ul>{features.map((feature) => <li key={feature}><span aria-hidden="true">✓</span>{feature}</li>)}</ul>
    <Link className="button inline-button" href="/">{plan.key === "Free" ? "เริ่มใช้งาน" : "แจ้งความสนใจ"}</Link>
  </article>;
}
