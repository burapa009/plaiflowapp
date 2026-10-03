import Link from "next/link";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import { sessionGET, sessionPOST } from "@/lib/session-api";
import { ConfirmButton, InviteForm, type InviteState } from "./invite-form";

type Member = { user_id: string; display_name: string; role: "Owner" | "Admin" | "Member" };
type Invitation = { id: string; expires_at: string };
type Plan = { name: string; limits: { members: number } };
const roleName = { Owner: "เจ้าของ", Admin: "ผู้ดูแล", Member: "สมาชิก" };

async function invite(organization: string, _state: InviteState, _form: FormData): Promise<InviteState> {
  "use server";
  void _state; void _form;
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/invitations`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  if (response?.status === 201) {
    const { token } = await response.json() as { token: string };
    return { path: `${(await headers()).get("origin")}/invite/accept?token=${encodeURIComponent(token)}` };
  }
  return { error: response?.status === 409 ? "สมาชิกและคำเชิญที่รอตอบรับครบจำนวนตามแพ็กเกจแล้ว" : response?.status === 403 ? "คุณไม่มีสิทธิ์เชิญสมาชิก" : "ยังสร้างคำเชิญไม่ได้ กรุณาลองอีกครั้ง" };
}

async function memberAction(organization: string, user: string, action: "role" | "remove" | "ownership", form: FormData) {
  "use server";
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const path = action === "ownership" ? `${base}/ownership` : `${base}/memberships/${encodeURIComponent(user)}/${action}`;
  const body = action === "role" ? new URLSearchParams({ role: String(form.get("role") ?? "") }) : action === "ownership" ? new URLSearchParams({ user_id: user }) : new URLSearchParams();
  const response = await sessionPOST(path, body);
  if (response?.status === 401) redirect("/");
  redirect(`/o/${encodeURIComponent(organization)}/members?${response?.ok ? "updated=1" : `error=${response?.status === 409 ? "conflict" : "change"}`}`);
}

async function revoke(organization: string, invitation: string) {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/invitations/${encodeURIComponent(invitation)}/revoke`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`/o/${encodeURIComponent(organization)}/members?${response?.ok ? "updated=1" : "error=change"}`);
}

async function leave(organization: string) {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/leave`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(response?.ok ? "/organizations" : `/o/${encodeURIComponent(organization)}/members?error=change`);
}

export default async function MembersPage({ params, searchParams }: { params: Promise<{ organization: string }>; searchParams: Promise<{ updated?: string; error?: string }> }) {
  const { organization } = await params;
  const query = await searchParams;
  const root = `/o/${encodeURIComponent(organization)}`;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const [membershipResponse, membersResponse, planResponse] = await Promise.all([sessionGET(base), sessionGET(`${base}/memberships`), sessionGET(`${base}/plan`)]);
  if ([membershipResponse, membersResponse, planResponse].some((response) => response?.status === 401)) redirect("/");
  if (!membershipResponse?.ok || !membersResponse?.ok || !planResponse?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดสมาชิกไม่ได้</h1><Link className="button inline-button" href={`${root}/members`}>ลองใหม่</Link></section>;
  const { membership } = await membershipResponse.json() as { membership: Member & { organization_name: string } };
  const { memberships } = await membersResponse.json() as { memberships: Member[] | null };
  const { effective_plan: plan } = await planResponse.json() as { effective_plan: Plan };
  const manager = membership.role === "Owner" || membership.role === "Admin";
  const inviteResponse = manager ? await sessionGET(`${base}/invitations`) : null;
  if (manager && !inviteResponse?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดคำเชิญไม่ได้</h1><Link className="button inline-button" href={`${root}/members`}>ลองใหม่</Link></section>;
  const { invitations } = inviteResponse ? await inviteResponse.json() as { invitations: Invitation[] | null } : { invitations: [] };
  const people = memberships ?? [];
  const pending = invitations ?? [];

  return <section className="workspace-page members-page"><div className="work-header"><div><p className="eyebrow">{membership.organization_name}</p><h1>สมาชิกและสิทธิ์</h1><p className="intro">ดูว่าใครเข้าถึงองค์กรนี้ได้ และจัดการสิทธิ์ตามบทบาทของคุณ</p></div><Link className="secondary-button" href="/organizations">เปลี่ยนองค์กร</Link></div>
    {query.updated && <p className="success-message" role="status">บันทึกการเปลี่ยนแปลงแล้ว</p>}
    {query.error && <p className="form-error" role="alert">{query.error === "conflict" ? "ทำรายการไม่ได้ในสถานะปัจจุบัน กรุณาตรวจสอบจำนวนสมาชิกหรือบทบาท" : "ยังทำรายการไม่ได้ กรุณาลองอีกครั้ง"}</p>}
    <div className="member-summary card"><strong>{people.length} / {plan.limits.members} สมาชิก</strong><span>แพ็กเกจ {plan.name}{manager && ` · ${pending.length} คำเชิญรอตอบรับ`}</span></div>
    <div className="member-layout"><article className="card member-list"><h2>สมาชิกปัจจุบัน</h2><ul>{people.map((person) => <li key={person.user_id}><div><strong>{person.display_name || "สมาชิก"}{person.user_id === membership.user_id && " (คุณ)"}</strong><span className="status-pill">{roleName[person.role]}</span></div><div className="card-actions">{membership.role === "Owner" && person.role !== "Owner" && person.user_id !== membership.user_id && <><form action={memberAction.bind(null, organization, person.user_id, "role")}><label className="sr-only" htmlFor={`role-${person.user_id}`}>สิทธิ์ของ {person.display_name}</label><select id={`role-${person.user_id}`} name="role" defaultValue={person.role}><option value="Member">สมาชิก</option><option value="Admin">ผู้ดูแล</option></select><button className="secondary-button" type="submit">บันทึกสิทธิ์</button></form><form action={memberAction.bind(null, organization, person.user_id, "ownership")}><ConfirmButton message={`โอนความเป็นเจ้าของให้ ${person.display_name || "สมาชิกนี้"} ใช่ไหม?`}>โอนเจ้าของ</ConfirmButton></form></>}{manager && person.user_id !== membership.user_id && person.role !== "Owner" && (membership.role === "Owner" || person.role === "Member") && <form action={memberAction.bind(null, organization, person.user_id, "remove")}><ConfirmButton message={`นำ ${person.display_name || "สมาชิกนี้"} ออกจากองค์กรใช่ไหม?`}>นำออก</ConfirmButton></form>}</div></li>)}</ul>{membership.role !== "Owner" && <form action={leave.bind(null, organization)}><ConfirmButton message="ออกจากองค์กรนี้ใช่ไหม? คุณจะเข้าถึงข้อมูลขององค์กรไม่ได้อีก">ออกจากองค์กร</ConfirmButton></form>}</article>
      {manager && <article className="card member-invites"><h2>เชิญสมาชิก</h2><p className="field-help">ลิงก์ใช้ได้ 24 ชั่วโมง ผู้รับจะเข้ามาเป็นสมาชิกก่อน เจ้าของปรับเป็นผู้ดูแลได้ภายหลัง</p>{people.length + pending.length >= plan.limits.members ? <p className="field-help">สิทธิ์สมาชิกเต็มแล้ว <Link href="/pricing">ดูแพ็กเกจ</Link> หรือยกเลิกคำเชิญที่ยังไม่ใช้</p> : <InviteForm action={invite.bind(null, organization)} />}{pending.length > 0 && <><h3>คำเชิญรอตอบรับ</h3><ul>{pending.map((item) => <li key={item.id}><span>หมดอายุ {new Date(item.expires_at).toLocaleString("th-TH", { timeZone: "Asia/Bangkok", dateStyle: "medium", timeStyle: "short" })}</span><form action={revoke.bind(null, organization, item.id)}><button className="secondary-button" type="submit">ยกเลิกคำเชิญ</button></form></li>)}</ul></>}</article>}
    </div>
  </section>;
}
