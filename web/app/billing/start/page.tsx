import Link from "next/link";
import { redirect } from "next/navigation";
import { sessionGET } from "@/lib/session-api";
import { isBillingInterval, type BillingInterval } from "@/lib/pricing";

type Organization = { id: string; name: string; role: string };

export default async function BillingStart({ searchParams }: { searchParams: Promise<{ plan?: string; interval?: string }> }) {
  const query = await searchParams;
  const plan = query.plan === "Starter" || query.plan === "Business" || query.plan === "Growth" ? query.plan : null;
  const interval: BillingInterval | null = isBillingInterval(query.interval) ? query.interval : null;
  if (!plan || !interval) redirect("/pricing");
  const [plansResponse, organizationsResponse] = await Promise.all([sessionGET("/v1/plans"), sessionGET("/v1/organizations")]);
  if (organizationsResponse?.status === 401) redirect("/");
  const catalog = plansResponse?.ok ? await plansResponse.json() as { billing_enabled: boolean } : null;
  const organizations = organizationsResponse?.ok ? (await organizationsResponse.json() as { organizations: Organization[] }).organizations : [];
  return <section className="workspace-page billing-page">
    <div className="work-header"><div><p className="eyebrow">PromptPay</p><h1>เลือก Organization ที่จะชำระ</h1><p className="intro">การชำระและสิทธิ์จะเป็นของ Organization ที่เลือก</p></div><Link className="secondary-button" href={`/pricing?interval=${interval}`}>กลับไปราคา</Link></div>
    {!catalog?.billing_enabled ? <p className="notice" role="status">ยังไม่เปิดรับชำระเงิน</p> : organizations.filter((item) => item.role === "Owner").length === 0 ? <p className="notice">ต้องเป็น Owner ของ Organization ก่อนจึงจะชำระเงินได้</p> : <div className="billing-stack">{organizations.filter((item) => item.role === "Owner").map((item) => <Link key={item.id} className="card billing-choice" href={`/o/${encodeURIComponent(item.id)}/billing?plan=${plan}&interval=${interval}`}><strong>{item.name}</strong><span>ตรวจสอบยอดและสร้าง QR PromptPay</span></Link>)}</div>}
  </section>;
}
