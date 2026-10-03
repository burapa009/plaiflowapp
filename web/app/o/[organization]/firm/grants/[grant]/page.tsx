import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { sessionGET, sessionPOST } from "@/lib/session-api";

type Grant = { id: string; firm_organization_id: string; client_organization_id: string; partner_name: string; status: string };
type Assignment = { user_id: string; display_name: string; assigned_at: string };
type Membership = { user_id: string; display_name?: string; role: "Owner" | "Admin" | "Member" };

async function removeAssignment(organization: string, grant: string, user: string) {
  "use server";
  const base = `/o/${encodeURIComponent(organization)}/firm/grants/${encodeURIComponent(grant)}`;
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/firm/grants/${encodeURIComponent(grant)}/assignments/${encodeURIComponent(user)}/remove`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`${base}?${response?.ok ? "updated=1" : "error=remove"}`);
}

export default async function GrantRosterPage({ params, searchParams }: {
  params: Promise<{ organization: string; grant: string }>;
  searchParams: Promise<{ updated?: string; error?: string }>;
}) {
  if (process.env.FIRM_ENABLED !== "true") notFound();
  const { organization, grant } = await params;
  const query = await searchParams;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const [orgResponse, grantsResponse, staffResponse] = await Promise.all([
    sessionGET(base), sessionGET(`${base}/firm/grants`),
    sessionGET(`${base}/firm/grants/${encodeURIComponent(grant)}/assignments`),
  ]);
  if ([orgResponse, grantsResponse, staffResponse].some((response) => response?.status === 401)) redirect("/");
  if (!orgResponse?.ok || !grantsResponse?.ok || !staffResponse?.ok) notFound();
  const { membership } = await orgResponse.json() as { membership: Membership };
  const { grants } = await grantsResponse.json() as { grants: Grant[] };
  const selected = grants.find((item) => item.id === grant);
  if (!selected) notFound();
  const { assignments } = await staffResponse.json() as { assignments: Assignment[] };
  const firmSide = selected.firm_organization_id === organization;
  const canManage = firmSide && (membership.role === "Owner" || membership.role === "Admin");
  const membersResponse = canManage ? await sessionGET(`${base}/memberships`) : null;
  const { memberships: members } = membersResponse?.ok ? await membersResponse.json() as { memberships: Membership[] } : { memberships: [] as Membership[] };
  const names = new Map(members.map((person) => [person.user_id, person.display_name || person.user_id]));

  return <section className="mx-auto max-w-[800px] space-y-5">
    <Link href={`/o/${encodeURIComponent(organization)}/firm`}>← กลับสำนักงานบัญชี</Link>
    <header className="work-header"><div><p className="eyebrow">Client access roster</p><h1>{selected.partner_name || (firmSide ? selected.client_organization_id : selected.firm_organization_id)}</h1>
      <p className="intro">สถานะความสัมพันธ์: {selected.status}</p></div></header>
    {query.updated && <p className="success-message" role="status">ถอนการมอบหมายแล้ว สิทธิ์เปิดข้อมูลหยุดทันที</p>}
    {query.error && <p className="form-error" role="alert">ถอนการมอบหมายไม่สำเร็จ ลองใหม่อีกครั้ง</p>}
    <section className="card"><h2>พนักงานที่ได้รับมอบหมาย</h2>
      {assignments.length === 0 ? <p>ยังไม่มีพนักงานที่ได้รับมอบหมาย</p> : <ul className="space-y-3">{assignments.map((person) =>
          <li key={person.user_id} className="card-actions"><span>{person.display_name || names.get(person.user_id) || person.user_id}</span>
          {canManage && <form action={removeAssignment.bind(null, organization, grant, person.user_id)}><button className="danger-button" type="submit">ถอนการมอบหมาย</button></form>}</li>)}</ul>}
    </section>
  </section>;
}
