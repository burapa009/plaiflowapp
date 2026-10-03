import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { sessionGET, sessionPOST } from "@/lib/session-api";

type Grant = {
  id: string; firm_organization_id: string; client_organization_id: string;
  partner_name: string; status: string; scopes: string[];
  request_expires_at: string; active_expires_at?: string;
};
type Membership = { user_id: string; display_name?: string; role: "Owner" | "Admin" | "Member" };
type PortfolioPage = { clients: { grant_id: string; client_organization_id: string; client_name: string; assigned: boolean; staff_count?: number }[]; next_cursor?: string };

async function requestClient(organization: string, form: FormData) {
  "use server";
  const base = `/o/${encodeURIComponent(organization)}/firm`;
  const client = String(form.get("client_organization_id") ?? "").trim();
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/firm/grants`, new URLSearchParams({ client_organization_id: client }));
  if (response?.status === 401) redirect("/");
  redirect(`${base}?${response?.ok ? "updated=1" : "error=request"}`);
}

async function transition(organization: string, grantID: string, action: string) {
  "use server";
  const base = `/o/${encodeURIComponent(organization)}/firm`;
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/firm/grants/${encodeURIComponent(grantID)}/${action}`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`${base}?${response?.ok ? "updated=1" : response?.status === 403 ? "error=recent" : "error=transition"}`);
}

async function assign(organization: string, grantID: string, form: FormData) {
  "use server";
  const base = `/o/${encodeURIComponent(organization)}/firm`;
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/firm/grants/${encodeURIComponent(grantID)}/assignments`,
    new URLSearchParams({ user_id: String(form.get("user_id") ?? "") }));
  if (response?.status === 401) redirect("/");
  redirect(`${base}?${response?.ok ? "updated=1" : "error=assign"}`);
}

export default async function FirmPage({ params, searchParams }: {
  params: Promise<{ organization: string }>;
  searchParams: Promise<{ updated?: string; error?: string; cursor?: string }>;
}) {
  if (process.env.FIRM_ENABLED !== "true") notFound();
  const { organization } = await params;
  const query = await searchParams;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const [orgResponse, grantResponse, planResponse] = await Promise.all([
    sessionGET(base), sessionGET(`${base}/firm/grants`), sessionGET(`${base}/plan`),
  ]);
  if (orgResponse?.status === 401 || grantResponse?.status === 401) redirect("/");
  if (!orgResponse?.ok || !grantResponse?.ok) return <section className="error-state" role="alert"><h1>เปิดพื้นที่สำนักงานไม่ได้</h1><p>ตรวจสอบสิทธิ์แล้วลองอีกครั้ง</p></section>;
  const { membership } = await orgResponse.json() as { membership: Membership };
  const { grants } = await grantResponse.json() as { grants: Grant[] };
  const { effective_plan } = planResponse?.ok ? await planResponse.json() as { effective_plan: { key: string } } : { effective_plan: { key: "Free" } };
  const manager = membership.role === "Owner" || membership.role === "Admin";
  const firmPlan = effective_plan.key === "AccountingFirm";
  const staffResponse = manager && firmPlan ? await sessionGET(`${base}/memberships`) : null;
  const { memberships: staff } = staffResponse?.ok ? await staffResponse.json() as { memberships: Membership[] } : { memberships: [] as Membership[] };
  const portfolioResponse = firmPlan ? await sessionGET(`${base}/firm/portfolio${query.cursor ? `?cursor=${encodeURIComponent(query.cursor)}` : ""}`) : null;
  const portfolio = portfolioResponse?.ok ? await portfolioResponse.json() as PortfolioPage : null;

  return <section className="mx-auto max-w-[1100px] space-y-6">
    <header className="work-header"><div><p className="eyebrow">Firm workspace</p><h1>สำนักงานบัญชี</h1>
      <p className="intro">ลูกค้าเป็นเจ้าของข้อมูลและอนุมัติสิทธิ์ให้สำนักงานเป็นรายแห่ง พนักงานต้องได้รับมอบหมายก่อนเปิดข้อมูล</p></div>
      <Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/tasks`}>งานของทีม</Link></header>
    {query.updated && <p className="success-message" role="status">บันทึกการเปลี่ยนแปลงแล้ว</p>}
    {query.error && <p className="form-error" role="alert">{query.error === "recent" ? "กรุณาลงชื่อเข้าใช้ใหม่ก่อนอนุมัติหรือเพิกถอนสิทธิ์" : "ดำเนินการไม่สำเร็จ ตรวจสอบสถานะและสิทธิ์แล้วลองใหม่"}</p>}
    {portfolio && <section className="card"><h2>ลูกค้าที่ดูแล</h2><p className="field-help">แสดงเฉพาะความสัมพันธ์ที่ยังใช้งานและพนักงานที่มีสิทธิ์ปัจจุบัน</p>
      <ul>{portfolio.clients.map((client) => <li key={client.grant_id}>{client.client_name || client.client_organization_id}{manager && <span> · พนักงาน {client.staff_count ?? 0} คน</span>}{!client.assigned && <span> · คุณยังไม่ได้รับมอบหมาย</span>}</li>)}</ul>
      {portfolio.next_cursor && <Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/firm?cursor=${encodeURIComponent(portfolio.next_cursor)}`}>หน้าถัดไป</Link>}
    </section>}
    {manager && firmPlan && <form className="card work-form" action={requestClient.bind(null, organization)}>
      <h2>ขอเชื่อมลูกค้า</h2><p className="field-help">ส่งรหัส Organization ให้ลูกค้าตรวจและอนุมัติ สิทธิ์เริ่มต้นคืออ่านอย่างเดียว</p>
      <label htmlFor="client-organization-id">รหัส Organization ลูกค้า</label>
      <input id="client-organization-id" name="client_organization_id" required pattern="[0-9a-fA-F-]{36}" autoComplete="off" />
      <button className="button" type="submit">ส่งคำขอ</button>
    </form>}
    <div className="space-y-4"><h2>ความสัมพันธ์ลูกค้า</h2>
      {grants.length === 0 && <p className="card">ยังไม่มีคำขอหรือความสัมพันธ์</p>}
      {grants.map((grant) => {
        const firmSide = grant.firm_organization_id === organization;
        const active = grant.status === "Active" && (!grant.active_expires_at || new Date(grant.active_expires_at) > new Date());
        return <article className="card" key={grant.id}>
          <h3>{grant.partner_name || (firmSide ? grant.client_organization_id : grant.firm_organization_id)}</h3>
          <p>สถานะ: {grant.status} · สิทธิ์: {grant.scopes.join(", ")}</p>
          {manager && <Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/firm/grants/${encodeURIComponent(grant.id)}`}>ดูทีมที่ได้รับมอบหมาย</Link>}
          {grant.active_expires_at && <p className="field-help">หมดอายุ: {new Date(grant.active_expires_at).toLocaleDateString("th-TH")}</p>}
          {manager && <div className="card-actions">
            {grant.status === "Requested" && !firmSide && <form action={transition.bind(null, organization, grant.id, "approve")}><button className="button" type="submit">อนุมัติสิทธิ์อ่านข้อมูล</button></form>}
            {grant.status === "ClientApproved" && firmSide && <form action={transition.bind(null, organization, grant.id, "accept")}><button className="button" type="submit">รับลูกค้า</button></form>}
            {["Requested", "ClientApproved"].includes(grant.status) && <form action={transition.bind(null, organization, grant.id, "cancel")}><button className="secondary-button" type="submit">ยกเลิกคำขอ</button></form>}
            {active && !firmSide && <form action={transition.bind(null, organization, grant.id, "revoke")}><button className="danger-button" type="submit">เพิกถอนสิทธิ์</button></form>}
            {active && firmSide && <form action={transition.bind(null, organization, grant.id, "end")}><button className="danger-button" type="submit">สิ้นสุดความสัมพันธ์</button></form>}
          </div>}
          {manager && firmSide && active && staff.length > 0 && <form className="work-form" action={assign.bind(null, organization, grant.id)}>
            <label htmlFor={`staff-${grant.id}`}>มอบหมายพนักงาน</label><select id={`staff-${grant.id}`} name="user_id" required>
              <option value="">เลือกพนักงาน</option>{staff.map((person) => <option key={person.user_id} value={person.user_id}>{person.display_name || person.user_id}</option>)}
            </select><button className="secondary-button" type="submit">มอบหมาย</button>
          </form>}
        </article>;
      })}
    </div>
  </section>;
}
