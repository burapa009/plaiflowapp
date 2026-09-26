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
    <div className="work-header"><div><p className="eyebrow">แพ็กเกจ PlaiFlow</p><h1>ราคาและรอบใช้งาน</h1><p className="intro">เลือกช่วงเวลาแล้วดูยอดเต็มที่ต้องชำระผ่าน PromptPay แต่ละรอบ</p></div><Link className="secondary-button" href="/">กลับหน้าหลัก</Link></div>
    <nav aria-label="เลือกรอบแพ็กเกจ" className="interval-tabs">{intervals.map((item) => <Link key={item} aria-current={item === interval ? "page" : undefined} href={`/pricing?interval=${item}`}>{intervalLabel(item)}</Link>)}</nav>
    {!catalog ? <div className="notice" role="alert"><h2>ยังโหลดแพ็กเกจไม่ได้</h2><p>กรุณาลองใหม่อีกครั้ง</p></div> : <>
      <div className="pricing-grid">{catalog.plans.filter((plan) => plan.key !== "AccountingFirm").map((plan) => <PlanCard key={plan.key} plan={plan} interval={interval} available={catalog.billing_enabled} />)}</div>
      {catalog.plans.filter((plan) => plan.key === "AccountingFirm").map((plan) => <div className="pricing-firm" key={plan.key}><PlanCard plan={plan} interval={interval} available={false} /></div>)}
      <p className="pricing-note">PromptPay ต้องสแกนและยืนยันการชำระใหม่ทุกงวด ไม่มีการหักเงินอัตโนมัติ · ผู้ขายยังไม่จด VAT จึงไม่คิด VAT เพิ่มจากราคาที่แสดง</p>
      {!catalog.billing_enabled && <p className="pricing-note">ยังไม่เปิดรับชำระเงินจริง</p>}
    </>}
  </section>;
}

function PlanCard({ plan, interval, available }: { plan: PlanDefinition; interval: BillingInterval; available: boolean }) {
  const price = plan.prices[interval];
  const features = Object.entries(plan.entitlements).filter(([key, enabled]) => enabled && featureLabels[key]).map(([key]) => featureLabels[key]);
  const firm = plan.key === "AccountingFirm";
  return <article className={`plan-card${plan.recommended ? " recommended" : ""}`}>
    {plan.recommended && <span className="status-pill">แนะนำ</span>}
    <h2>{plan.name}</h2>
    <p className="plan-price"><strong>{formatSatang(price.total_satang)}</strong><span> / {intervalLabel(interval)}</span></p>
    {plan.key !== "Free" && <p className="field-help">ยอดเต็มสำหรับ {intervalLabel(interval)} · เมื่อครบกำหนด ต้องชำระรอบใหม่ด้วยตนเอง</p>}
    {price.months > 1 && <p className="field-help">เฉลี่ย {formatSatang(price.effective_month_satang)} / เดือน{price.saving_percent > 0 ? ` · ประหยัด ${price.saving_percent}%` : ""}</p>}
    <ul>
      <li><span aria-hidden="true">✓</span>{plan.limits.members} {firm ? "ที่นั่งสำนักงาน" : "สมาชิก"}</li>
      <li><span aria-hidden="true">✓</span>{plan.limits.line_groups} LINE groups</li>
      <li><span aria-hidden="true">✓</span>{plan.limits.documents_per_month.toLocaleString("th-TH")} เอกสารต่อเดือน</li>
      <li><span aria-hidden="true">✓</span>เครดิตสแกน OCR รวม 0 หน้า</li>
      {firm && <li><span aria-hidden="true">✓</span>{plan.limits.client_relationships} ความสัมพันธ์ลูกค้า</li>}
      {(firm ? ["จัดการสำนักงานบัญชี", "พอร์ตลูกค้า", "ตรวจทานและส่งออกเอกสารที่อนุมัติ"] : features).map((feature) => <li key={feature}><span aria-hidden="true">✓</span>{feature}</li>)}
    </ul>
    {firm ? <span className="secondary-button inline-button" aria-disabled="true">ยังไม่เปิดจำหน่าย</span> : plan.key === "Free" ? <Link className="button inline-button" href="/organizations">เริ่มใช้งานฟรี</Link> : available ? <Link className="button inline-button" href={`/billing/start?plan=${plan.key}&interval=${interval}`}>เลือกแพ็กเกจ</Link> : <span className="secondary-button inline-button" aria-disabled="true">รอเปิดชำระเงิน</span>}
  </article>;
}
