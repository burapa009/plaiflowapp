import { cookies } from "next/headers";
import Link from "next/link";
import { redirect } from "next/navigation";
import { sessionGET, sessionPOST } from "@/lib/session-api";

type Membership = { role: "Owner" | "Admin" | "Member" };
type Connection = { status: "Not Connected" | "Connected" | "Reauthorization Required"; google_email?: string; folder_id?: string };
type Plan = { key: string; name: string; entitlements: Record<string, boolean> };

async function connectDrive(organization: string) {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/drive/connect`, new URLSearchParams(), "manual");
  if (response?.status === 401) redirect("/");
  if (response?.status !== 303) redirect(`/o/${encodeURIComponent(organization)}/connections?error=${response?.status === 402 ? "plan" : response?.status === 403 ? "recent" : "connect"}`);
  const cookieHeader = response.headers.get("set-cookie") ?? "";
  const match = cookieHeader.match(/__Host-plaiflow-drive=([^;]+)/);
  const location = response.headers.get("location");
  if (!match || !location) redirect(`/o/${encodeURIComponent(organization)}/connections?error=connect`);
  (await cookies()).set("__Host-plaiflow-drive", match[1], { httpOnly: true, secure: true, sameSite: "lax", path: "/", maxAge: 600 });
  redirect(location);
}

async function disconnectDrive(organization: string) {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/drive/disconnect`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`/o/${encodeURIComponent(organization)}/connections?${response?.ok ? "drive=disconnected" : "error=disconnect"}`);
}

async function checkDrive(organization: string) {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/drive/check`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  const body = response?.ok ? null : await response?.json().catch(() => ({})) as { code?: string } | undefined;
  redirect(`/o/${encodeURIComponent(organization)}/connections?${response?.ok ? "drive=checked" : body?.code === "reconnect_required" ? "drive=reconnect" : "error=check"}`);
}

export default async function ConnectionsPage({ params, searchParams }: { params: Promise<{ organization: string }>; searchParams: Promise<{ drive?: string; error?: string }> }) {
  const { organization } = await params;
  const query = await searchParams;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const [membershipResponse, driveResponse, planResponse] = await Promise.all([sessionGET(base), sessionGET(`${base}/drive`), sessionGET(`${base}/plan`)]);
  if ([membershipResponse, driveResponse, planResponse].some((response) => response?.status === 401)) redirect("/");
  if (!membershipResponse?.ok || !driveResponse?.ok || !planResponse?.ok) return <section className="error-state" role="alert"><h1>ยังเปิด Connections ไม่ได้</h1><Link className="button inline-button" href={`/o/${encodeURIComponent(organization)}/connections`}>ลองใหม่</Link></section>;
  const { membership } = await membershipResponse.json() as { membership: Membership };
  const { configured, connection } = await driveResponse.json() as { configured: boolean; connection: Connection };
  const { effective_plan } = await planResponse.json() as { effective_plan: Plan };
  const manager = membership.role === "Owner" || membership.role === "Admin";
  const included = effective_plan.entitlements["business_contacts.export.drive"] === true;

  return <section className="connections-page">
    <div className="work-header"><div><p className="eyebrow">Authorized settings</p><h1>Connections</h1><p className="intro">Google Drive ใช้การอนุญาตแยกจาก Google Login และขอสิทธิ์เฉพาะไฟล์ที่ PlaiFlow สร้าง</p></div><div className="card-actions"><Link className="secondary-button" href="/pricing">ดูแพ็กเกจ</Link><Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/tasks`}>กลับพื้นที่งาน</Link></div></div>
    {query.drive && <p className="success-message" role="status">{query.drive === "connected" ? "เชื่อมต่อ Google Drive แล้ว" : query.drive === "disconnected" ? "ยกเลิกการเชื่อมต่อแล้ว PlaiFlow หยุดใช้สิทธิ์เดิมทันที" : query.drive === "checked" ? "ตรวจสอบสิทธิ์แล้ว" : "สิทธิ์หมดอายุ กรุณาเชื่อมต่อใหม่ ระบบสร้างงานแจ้ง Owner แล้ว"}</p>}
    {query.error && <p className="form-error" role="alert">{connectionError(query.error)}</p>}
    <article className="card connection-card"><div><p className="field-help">Google Drive · แพ็กเกจ {effective_plan.name}</p><h2>{connection.status}</h2>{connection.google_email && <p>{connection.google_email}</p>}<p className="field-help">ขอบเขตสิทธิ์: drive.file, email และ openid เท่านั้น</p></div>
      {!manager ? <p className="field-help">เฉพาะ Owner หรือ Admin ของลูกค้าที่เปลี่ยนการเชื่อมต่อได้</p> : !configured ? <p className="form-error">Staging ยังไม่ได้ตั้งค่า Google Drive OAuth</p> : !included ? <div><p className="field-help">แพ็กเกจนี้ยังไม่รวม Google Drive</p><Link className="secondary-button" href="/pricing">ดูแพ็กเกจ</Link></div> : connection.status === "Connected" ? <div className="card-actions"><form action={checkDrive.bind(null, organization)}><button className="secondary-button" type="submit">ตรวจสอบสิทธิ์</button></form><form action={disconnectDrive.bind(null, organization)}><button className="danger-button" type="submit">ยกเลิกการเชื่อมต่อ</button></form></div> : <form action={connectDrive.bind(null, organization)}><button className="button" type="submit">{connection.status === "Reauthorization Required" ? "เชื่อมต่อใหม่" : "เชื่อมต่อ Google Drive"}</button></form>}
    </article>
  </section>;
}

function connectionError(code: string) { return code === "plan" ? "แพ็กเกจปัจจุบันยังไม่รวม Google Drive" : code === "recent" ? "เพื่อความปลอดภัย กรุณาล็อกอินใหม่แล้วลองเชื่อมต่ออีกครั้ง" : code === "disconnect" ? "ยังยกเลิกการเชื่อมต่อไม่ได้ กรุณาลองใหม่" : code === "check" ? "ตรวจสอบสิทธิ์ไม่สำเร็จ กรุณาลองใหม่" : "ยังเริ่มเชื่อมต่อ Google Drive ไม่ได้"; }
