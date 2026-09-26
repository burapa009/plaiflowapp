import Link from "next/link";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { sessionGET, sessionPOST } from "@/lib/session-api";
import { CodeForm, type CodeState } from "./code-form";

type Membership = { role: "Owner" | "Admin" | "Member"; organization_name: string };
type Connection = { id: string; group_id: string; status: "connected" | "disconnected"; connected_at: string };
type Plan = { name: string; limits: { line_groups: number } };
type Identity = { provider: string; linked: boolean };

async function createCode(organization: string, _state: CodeState, _form: FormData): Promise<CodeState> {
  "use server";
  void _state; void _form;
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/line-link-codes`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  if (response?.status === 201) {
    const code = await response.json() as { command: string; expires_at: string };
    return { command: code.command, expiresAt: code.expires_at };
  }
  const body = await response?.json().catch(() => ({})) as { code?: string } | undefined;
  return { error: response?.status === 409 ? "จำนวนกลุ่ม LINE เต็มตามแพ็กเกจแล้ว ยกเลิกกลุ่มเดิมหรือดูแพ็กเกจอื่น" : body?.code === "recent_auth_required" ? "กรุณายืนยันตัวตนใหม่ก่อนสร้างโค้ด" : response?.status === 403 ? "บัญชี LINE ยังไม่เชื่อมกับผู้ใช้ของคุณ หรือคุณไม่มีสิทธิ์จัดการกลุ่ม" : "ยังสร้างโค้ดไม่ได้ กรุณาลองอีกครั้ง" };
}

async function disconnect(organization: string, connection: string) {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/line-connections/${encodeURIComponent(connection)}/disconnect`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`/o/${encodeURIComponent(organization)}/line-groups?${response?.ok ? "disconnected=1" : "error=disconnect"}`);
}

export default async function LINEGroupsPage({ params, searchParams }: { params: Promise<{ organization: string }>; searchParams: Promise<{ disconnected?: string; error?: string }> }) {
  const { organization } = await params;
  const query = await searchParams;
  const root = `/o/${encodeURIComponent(organization)}`;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const [membershipResponse, groupsResponse, planResponse, sessionResponse] = await Promise.all([
    sessionGET(base), sessionGET(`${base}/line-connections`), sessionGET(`${base}/plan`), sessionGET("/v1/session"),
  ]);
  if ([membershipResponse, groupsResponse, planResponse, sessionResponse].some((response) => response?.status === 401)) redirect("/");
  if (!membershipResponse?.ok || !groupsResponse?.ok || !planResponse?.ok || !sessionResponse?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดกลุ่ม LINE ไม่ได้</h1><Link className="button inline-button" href={`${root}/line-groups`}>ลองใหม่</Link></section>;
  const { membership } = await membershipResponse.json() as { membership: Membership };
  const { connections } = await groupsResponse.json() as { connections: Connection[] | null };
  const { effective_plan: plan } = await planResponse.json() as { effective_plan: Plan };
  const { identities, recent_auth: recentAuth } = await sessionResponse.json() as { identities: Identity[] | null; recent_auth: boolean };
  const active = (connections ?? []).filter((connection) => connection.status === "connected");
  const manager = membership.role === "Owner" || membership.role === "Admin";
  const linkedLINE = (identities ?? []).some((identity) => identity.provider === "line" && identity.linked);
  const reauthProvider = linkedLINE ? "line" : (identities ?? []).find((identity) => identity.linked)?.provider;
  const csrf = (await cookies()).get("__Host-plaiflow-csrf")?.value;
  const returnTo = encodeURIComponent(`${root}/line-groups`);

  return <section className="workspace-page line-groups-page">
    <div className="work-header"><div><p className="eyebrow">{membership.organization_name}</p><h1>กลุ่ม LINE</h1><p className="intro">เชื่อมกลุ่มเพื่อรับเอกสารเข้าพื้นที่ขององค์กรนี้ เฉพาะสมาชิกองค์กรที่เชื่อมบัญชี LINE แล้วจึงส่งเอกสารได้ การอยู่ในกลุ่มไม่ให้สิทธิ์อ่านข้อมูลใน PlaiFlow</p></div><Link className="secondary-button" href="/organizations">เปลี่ยนองค์กร</Link></div>
    {query.disconnected && <p className="success-message" role="status">ยกเลิกการเชื่อมต่อแล้ว ระบบจะหยุดรับเอกสารใหม่จากกลุ่มนี้</p>}
    {query.error && <p className="form-error" role="alert">ยังยกเลิกการเชื่อมต่อไม่ได้ กรุณาลองอีกครั้ง</p>}
    <div className="line-group-summary card"><strong>{active.length} / {plan.limits.line_groups} กลุ่ม</strong><span>แพ็กเกจ {plan.name} · สิทธิ์ที่เหลือ {Math.max(0, plan.limits.line_groups - active.length)} กลุ่ม</span></div>
    <div className="line-groups-layout"><article className="card line-group-setup"><h2>เชื่อมกลุ่มใหม่</h2>
      <ol><li>เชื่อมบัญชี LINE ของคุณกับ PlaiFlow และเพิ่มบัญชีทางการ PlaiFlow เข้ากลุ่ม</li><li>สร้างโค้ดเชื่อมจากหน้านี้</li><li>ส่งข้อความโค้ดในกลุ่มจากแอป LINE บนมือถือด้วยบัญชีเดียวกัน แล้วกดตรวจสถานะ</li></ol>
      <p className="field-help">หากเพิ่มบัญชีทางการเข้ากลุ่มไม่ได้ ให้ผู้ดูแล LINE Official Account เปิดสิทธิ์ให้บอตเข้ากลุ่มก่อน</p>
      {!manager ? <p className="field-help">เฉพาะเจ้าของหรือผู้ดูแลจัดการกลุ่มได้</p> : active.length >= plan.limits.line_groups ? <p className="field-help">ใช้สิทธิ์ครบแล้ว <Link href="/pricing">ดูแพ็กเกจ</Link> หรือยกเลิกกลุ่มเดิมก่อน</p> : !recentAuth && reauthProvider && csrf ? <form method="post" action={`/api/auth/${reauthProvider}/reauth?return_to=${returnTo}`}><input type="hidden" name="csrf_token" value={csrf} /><button className="button" type="submit">ยืนยันตัวตนอีกครั้ง</button></form> : !linkedLINE && csrf ? <form method="post" action={`/api/auth/line/link?return_to=${returnTo}`}><input type="hidden" name="csrf_token" value={csrf} /><button className="button" type="submit">เชื่อมบัญชี LINE ของฉัน</button></form> : <CodeForm action={createCode.bind(null, organization)} />}
      <Link className="secondary-button inline-button" href={`${root}/line-groups`}>ตรวจสถานะการเชื่อม</Link>
    </article><article className="card line-group-list"><h2>กลุ่มที่เชื่อมแล้ว</h2>{active.length === 0 ? <p className="field-help">ยังไม่มีกลุ่มที่เชื่อม ทำตาม 3 ขั้นตอนทางซ้ายเพื่อเริ่มใช้งาน</p> : <ul>{active.map((connection, index) => <li key={connection.id}><div><strong>กลุ่ม LINE {index + 1}</strong><span className="field-help">รหัสกลุ่มลงท้าย {connection.group_id.slice(-6)} · เชื่อมเมื่อ {new Date(connection.connected_at).toLocaleString("th-TH", { timeZone: "Asia/Bangkok", dateStyle: "medium", timeStyle: "short" })}</span></div>{manager && <form action={disconnect.bind(null, organization, connection.id)}><button className="danger-button" type="submit">ยกเลิกการเชื่อม</button></form>}</li>)}</ul>}</article></div>
  </section>;
}
