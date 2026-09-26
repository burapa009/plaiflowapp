import Link from "next/link";
import { getAPIBaseURL } from "@/lib/api-config";
import { formatSatang, intervalLabel, isBillingInterval, type BillingInterval, type PlanDefinition } from "@/lib/pricing";
import { PlanArtwork } from "./plan-artwork";

const intervals: BillingInterval[] = ["monthly", "six_months", "yearly"];
const scanPacks = [
  { pages: 500, satang: 21900 },
  { pages: 800, satang: 33900 },
  { pages: 1800, satang: 69900 },
  { pages: 5000, satang: 189900 },
];
const descriptions: Record<PlanDefinition["key"], string> = {
  Free: "เริ่มต้นใช้งานได้ฟรี เหมาะสำหรับผู้เริ่มต้น",
  Starter: "เหมาะสำหรับธุรกิจขนาดเล็ก เริ่มจัดการข้อมูลได้มากขึ้น",
  Business: "เหมาะสำหรับธุรกิจที่ต้องการขยายการใช้งาน",
  Growth: "เหมาะสำหรับธุรกิจที่มีทีมและเอกสารมากขึ้น",
  AccountingFirm: "สำหรับสำนักงานบัญชีที่ดูแลหลายองค์กร",
};
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
    <div className="work-header pricing-header"><div><p className="eyebrow">แพ็กเกจ PLAIFLOW</p><h1>ราคาและรอบใช้งาน</h1><p className="intro">เลือกช่วงเวลาแล้วดูยอดเต็มที่ต้องชำระผ่าน PromptPay แต่ละรอบ</p></div><Link className="secondary-button pricing-home" href="/"><svg aria-hidden="true" viewBox="0 0 24 24" width="19" height="19" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m3 10 9-7 9 7v11h-6v-7H9v7H3z" /></svg>กลับหน้าหลัก</Link></div>
    <nav aria-label="เลือกรอบแพ็กเกจ" className="interval-tabs pricing-tabs">{intervals.map((item) => <Link key={item} aria-current={item === interval ? "page" : undefined} href={`/pricing?interval=${item}`}>{intervalLabel(item)}</Link>)}</nav>
    {!catalog ? <div className="notice" role="alert"><h2>ยังโหลดแพ็กเกจไม่ได้</h2><p>กรุณาลองใหม่อีกครั้ง</p></div> : <>
      <div className="pricing-grid">{catalog.plans.filter((plan) => plan.key !== "AccountingFirm").map((plan) => <PlanCard key={plan.key} plan={plan} interval={interval} available={catalog.billing_enabled} />)}</div>
      <p className="pricing-note">PromptPay ต้องชำระใหม่ทุกงวด ไม่มีการหักเงินอัตโนมัติ · ราคาที่แสดงเป็นยอดเต็มต่อรอบ</p>
      {!catalog.billing_enabled && <p className="pricing-note">ยังไม่เปิดรับชำระเงินจริง</p>}
      <section className="scan-packs" aria-labelledby="scan-packs-title"><div className="scan-packs-heading"><div><p className="eyebrow">เครดิตสแกนเอกสาร</p><h2 id="scan-packs-title">ซื้อเครดิตเพิ่มเมื่อพร้อมใช้งาน</h2><p className="intro">1 ภาพหรือ 1 หน้า PDF ใช้ 1 เครดิต · เครดิตที่ซื้อไม่หมดอายุและไม่เพิ่มโควตารับเอกสาร</p></div><span className="scan-status">รอเปิดใช้ OCR</span></div><div className="scan-pack-grid">{scanPacks.map((pack) => <article className="scan-pack" key={pack.pages}><strong>{pack.pages.toLocaleString("th-TH")} หน้า</strong><p>{formatSatang(pack.satang)} <span>/ ซื้อครั้งเดียว</span></p><span className="secondary-button inline-button" aria-disabled="true">รอเปิดซื้อเครดิต</span></article>)}</div><p className="field-help">ยังไม่รับชำระค่าเครดิตสแกน จนกว่าระบบ OCR จะผ่านการทดสอบความแม่นยำและรองรับปริมาณงาน</p></section>
      {catalog.plans.filter((plan) => plan.key === "AccountingFirm").map((plan) => <section className="pricing-firm" key={plan.key} aria-label="แพ็กเกจสำนักงานบัญชี"><PlanCard plan={plan} interval={interval} available={false} /></section>)}
    </>}
  </section>;
}

function PlanCard({ plan, interval, available }: { plan: PlanDefinition; interval: BillingInterval; available: boolean }) {
  const price = plan.prices[interval];
  const features = Object.entries(plan.entitlements).filter(([key, enabled]) => enabled && featureLabels[key]).map(([key]) => featureLabels[key]);
  const firm = plan.key === "AccountingFirm";
  return <article className={`plan-card plan-${plan.key.toLowerCase()}${plan.recommended ? " recommended" : ""}`}>
    {plan.recommended && <span className="plan-ribbon"><svg aria-hidden="true" viewBox="0 0 24 24" width="17" height="17" fill="currentColor"><path d="m12 2 2.9 6.3 6.8.8-5 4.7 1.3 6.7-6-3.3-6 3.3 1.3-6.7-5-4.7 6.8-.8z" /></svg>แนะนำ</span>}
    <div className="plan-card-copy"><h2>{plan.name}</h2><p className="plan-description">{descriptions[plan.key]}</p><p className="plan-price"><strong>{formatSatang(price.total_satang)}</strong><span> / {intervalLabel(interval)}</span></p>{plan.key !== "Free" && <p className="plan-period-note">ยอดเต็มสำหรับ {intervalLabel(interval)} · เมื่อครบกำหนดต้องชำระรอบใหม่ด้วยตนเอง</p>}{price.months > 1 && <p className="plan-saving">เฉลี่ย {formatSatang(price.effective_month_satang)} / เดือน · ประหยัด {price.saving_percent}%</p>}</div>
    {!firm && <PlanArtwork name={plan.key} />}
    <ul className="plan-features">
      <li><span aria-hidden="true">✓</span>{plan.limits.members} {firm ? "ที่นั่งสำนักงาน" : "สมาชิก"}</li>
      <li><span aria-hidden="true">✓</span>{plan.limits.line_groups} LINE groups</li>
      <li><span aria-hidden="true">✓</span>{plan.limits.documents_per_month.toLocaleString("th-TH")} เอกสารต่อเดือน</li>
      <li><span aria-hidden="true">✓</span>เครดิตสแกน OCR รวม 0 หน้า</li>
      {firm && <li><span aria-hidden="true">✓</span>{plan.limits.client_relationships} ความสัมพันธ์ลูกค้า</li>}
      {(firm ? ["จัดการสำนักงานบัญชี", "พอร์ตลูกค้า", "ตรวจทานและส่งออกเอกสารที่อนุมัติ"] : features).map((feature) => <li key={feature}><span aria-hidden="true">✓</span>{feature}</li>)}
    </ul>
    {firm ? <span className="secondary-button inline-button plan-action" aria-disabled="true">ยังไม่เปิดจำหน่าย</span> : plan.key === "Free" ? <Link className="button inline-button plan-action" href="/organizations">เริ่มใช้งานฟรี <ArrowIcon /></Link> : available ? <Link className="button inline-button plan-action" href={`/billing/start?plan=${plan.key}&interval=${interval}`}>เลือกแพ็กเกจ <ArrowIcon /></Link> : <span className="secondary-button inline-button plan-action" aria-disabled="true">รอเปิดชำระเงิน <ArrowIcon /></span>}
  </article>;
}

function ArrowIcon() { return <svg aria-hidden="true" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d="m9 5 7 7-7 7" /></svg>; }
