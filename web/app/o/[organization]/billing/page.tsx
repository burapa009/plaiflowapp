import Image from "next/image";
import Link from "next/link";
import { redirect } from "next/navigation";
import { sessionGET, sessionPOST } from "@/lib/session-api";
import { formatSatang, intervalLabel, isBillingInterval, type BillingInterval, type PlanDefinition } from "@/lib/pricing";

type Intent = { id: string; plan: string; interval: BillingInterval; amount_satang: number; status: string; qr_image_url?: string; expires_at: string };
type Summary = { plan: string; interval?: BillingInterval; status: "free" | "trial" | "manual" | "active" | "canceling" | "grace" | "expired"; paid_through?: string; grace_until?: string; cancel_at_period_end: boolean; pending?: Intent; pending_expired?: boolean; latest_payment_status?: string };
type Catalog = { plans: PlanDefinition[]; billing_enabled: boolean };

function date(value: string) { return new Intl.DateTimeFormat("th-TH", { dateStyle: "long", timeZone: "Asia/Bangkok" }).format(new Date(value)); }

async function createQR(organization: string, form: FormData) {
  "use server";
  const target = `/o/${encodeURIComponent(organization)}/billing`;
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/billing/intents`, new URLSearchParams({ plan: String(form.get("plan") ?? ""), interval: String(form.get("interval") ?? "") }));
  if (response?.status === 401) redirect("/");
  redirect(`${target}${response?.ok ? "?payment=pending" : `?error=${response?.status === 409 ? "pending" : "checkout"}`}`);
}

async function changeCancellation(organization: string, cancel: boolean) {
  "use server";
  const target = `/o/${encodeURIComponent(organization)}/billing`;
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/billing/${cancel ? "cancel" : "resume"}`, new URLSearchParams(cancel ? { confirm: "cancel" } : {}));
  if (response?.status === 401) redirect("/");
  redirect(`${target}${response?.ok ? "?updated=1" : `?error=${response?.status === 403 ? "auth" : "change"}`}`);
}

export default async function BillingPage({ params, searchParams }: { params: Promise<{ organization: string }>; searchParams: Promise<{ plan?: string; interval?: string; payment?: string; error?: string; updated?: string }> }) {
  const { organization } = await params;
  const query = await searchParams;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const [summaryResponse, catalogResponse, membershipResponse] = await Promise.all([sessionGET(`${base}/billing`), sessionGET("/v1/plans"), sessionGET(base)]);
  if (summaryResponse?.status === 401 || membershipResponse?.status === 401) redirect("/");
  if (summaryResponse?.status === 403) return <section className="error-state" role="alert"><h1>เฉพาะ Owner ที่ดูข้อมูลการชำระเงินได้</h1><Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/tasks`}>กลับไปพื้นที่ทำงาน</Link></section>;
  if (!summaryResponse?.ok || !catalogResponse?.ok || !membershipResponse?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดข้อมูลการชำระเงินไม่ได้</h1><p>ระบบชำระเงินอาจยังไม่เปิดใช้</p><Link className="secondary-button" href="/pricing">ดูราคา</Link></section>;
  const summary = await summaryResponse.json() as Summary;
  const catalog = await catalogResponse.json() as Catalog;
  const { membership } = await membershipResponse.json() as { membership: { organization_name: string; role: string } };
  if (!catalog.billing_enabled || membership.role !== "Owner") return <section className="error-state"><h1>ยังไม่เปิดรับชำระเงิน</h1><Link href="/pricing">ดูราคา</Link></section>;
  const selectedPlan = query.plan === "Starter" || query.plan === "Business" ? query.plan : summary.plan === "Starter" || summary.plan === "Business" ? summary.plan : "Starter";
  const selectedInterval = isBillingInterval(query.interval) ? query.interval : summary.interval ?? "monthly";
  const chosen = catalog.plans.find((item) => item.key === selectedPlan);
  const price = chosen?.prices[selectedInterval];
  const pending = summary.pending;
  const qrReady = pending?.status === "awaiting_payment" && pending.qr_image_url?.startsWith("https://api.omise.co/charges/") && !summary.pending_expired;
  const statusText = { free: "แพ็กเกจฟรี", trial: "ช่วงทดลองใช้", manual: "สิทธิ์เดิมที่ให้ไว้", active: "ชำระแล้ว", canceling: "ยกเลิกเมื่อครบกำหนด", grace: "เลยกำหนด อยู่ในช่วงผ่อนผัน", expired: "สิทธิ์ชำระเงินสิ้นสุดแล้ว" }[summary.status];
  const canBuy = !pending && (summary.status === "free" || summary.status === "trial" || summary.status === "expired" || summary.status === "grace" || (summary.status === "active" && summary.plan === selectedPlan && summary.interval === selectedInterval));
  return <section className="workspace-page billing-page">
    <div className="work-header"><div><p className="eyebrow">การชำระเงิน · {membership.organization_name}</p><h1>แพ็กเกจและ PromptPay</h1><p className="intro">ชำระด้วย QR ใหม่ทุกงวด ไม่มีการหักเงินอัตโนมัติ</p></div><Link className="secondary-button" href="/pricing">เปรียบเทียบราคา</Link></div>
    {query.payment && <p className="notice" role="status">สร้าง QR แล้ว สิทธิ์จะเปลี่ยนหลังผู้ให้บริการยืนยันการชำระเท่านั้น</p>}
    {query.updated && <p className="success-message" role="status">บันทึกการเปลี่ยนแปลงแล้ว</p>}
    {query.error && <p className="form-error" role="alert">{query.error === "pending" ? "มีรายการชำระค้างอยู่ กรุณาตรวจสอบก่อนสร้างรายการใหม่" : query.error === "auth" ? "กรุณาเข้าสู่ระบบใหม่ก่อนเปลี่ยนการยกเลิก" : "ยังทำรายการไม่ได้ กรุณาลองใหม่"}</p>}
    <div className="billing-stack">
      {(summary.latest_payment_status === "failed" || summary.latest_payment_status === "expired") && <p className="form-error" role="status">รายการ PromptPay ล่าสุดไม่สำเร็จหรือหมดอายุแล้ว คุณสามารถสร้าง QR ใหม่ได้ สิทธิ์ที่ชำระไว้ก่อนหน้ายังคงอยู่จนถึงวันสิ้นสุดที่แสดงด้านล่าง</p>}
      <article className="card"><h2>แพ็กเกจปัจจุบัน</h2><p><strong>{statusText}</strong> · {summary.plan}</p>{summary.interval && <p>รอบ {intervalLabel(summary.interval)}</p>}{summary.paid_through && <p>สิทธิ์ชำระแล้วถึง {date(summary.paid_through)}</p>}{summary.status === "grace" && summary.grace_until && <p className="form-error">ชำระรอบใหม่ก่อน {date(summary.grace_until)} เพื่อคงสิทธิ์แพ็กเกจ</p>}{summary.status === "canceling" && <p>เมื่อครบกำหนด สิทธิ์แพ็กเกจจะสิ้นสุดและไม่ส่งคำเตือนให้ต่ออายุ</p>}</article>
      {pending && <article className="card"><h2>รายการ PromptPay ที่รอชำระ</h2><p><strong>{formatSatang(pending.amount_satang)}</strong> สำหรับ {pending.plan} รอบ{intervalLabel(pending.interval)}</p>{qrReady ? <><Image unoptimized src={pending.qr_image_url!} alt="QR PromptPay สำหรับชำระรายการนี้" width={260} height={260} className="billing-qr" /><p>QR หมดอายุ {date(pending.expires_at)} · ตรวจชื่อผู้รับและยอดในแอปธนาคารก่อนยืนยัน</p><Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/billing`}>ตรวจสถานะอีกครั้ง</Link></> : <p>QR ยังไม่พร้อมหรือหมดอายุ กรุณารอระบบตรวจสอบหรือกลับมาสร้างรายการใหม่</p>}</article>}
      {!pending && chosen && price && <article className="card"><h2>ชำระรอบใหม่</h2><nav aria-label="เลือกรอบแพ็กเกจ" className="interval-tabs">{(["monthly", "six_months", "yearly"] as const).map((item) => <Link key={item} aria-current={item === selectedInterval ? "page" : undefined} href={`?plan=${selectedPlan}&interval=${item}`}>{intervalLabel(item)}</Link>)}</nav><p className="billing-full-charge"><strong>{formatSatang(price.total_satang)}</strong> / {intervalLabel(selectedInterval)}</p><p>ยอดนี้เป็นยอดเต็มที่ชำระผ่าน QR ครั้งนี้ เมื่อครบช่วงใช้งานต้องชำระใหม่ด้วยตนเอง</p>{price.months > 1 && <p className="field-help">เฉลี่ย {formatSatang(price.effective_month_satang)} ต่อเดือนเพื่อเปรียบเทียบ · ไม่ใช่ยอดที่หักรายเดือน</p>}{canBuy ? <form action={createQR.bind(null, organization)}><input type="hidden" name="plan" value={selectedPlan} /><input type="hidden" name="interval" value={selectedInterval} /><button className="button" type="submit">สร้าง QR PromptPay {formatSatang(price.total_satang)}</button></form> : <p className="notice">การเปลี่ยนแพ็กเกจระหว่างรอบยังไม่เปิดให้ทำจากหน้านี้</p>}</article>}
      {summary.status === "active" && <details className="card"><summary>ยกเลิกการต่ออายุ</summary><p>สิทธิ์ยังใช้ได้ถึง {summary.paid_through && date(summary.paid_through)} การยกเลิกจะหยุดคำเตือนต่ออายุ และไม่มีการหักเงินอัตโนมัติอยู่แล้ว</p><form action={changeCancellation.bind(null, organization, true)}><button className="danger-button" type="submit">ยืนยันยกเลิกเมื่อครบกำหนด</button></form></details>}
      {summary.status === "canceling" && <form action={changeCancellation.bind(null, organization, false)} className="card"><p>ต้องการรับคำเตือนให้ชำระรอบใหม่อีกครั้ง?</p><button className="secondary-button" type="submit">กลับมาใช้งานต่อ</button></form>}
    </div>
  </section>;
}
