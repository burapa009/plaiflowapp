import Link from "next/link";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { sessionGET, sessionPOST } from "@/lib/session-api";

const inviteCookie = "__Host-plaiflow-invite";

async function signIn(token: string, provider: "line" | "google") {
  "use server";
  if (!token || token.length > 256) redirect("/invite/accept?error=invalid");
  (await cookies()).set(inviteCookie, token, { httpOnly: true, secure: true, sameSite: "lax", path: "/", maxAge: 24 * 60 * 60 });
  redirect(`/api/auth/${provider}/start?return_to=/invite/accept`);
}

async function accept(token: string) {
  "use server";
  if (!token || token.length > 256) redirect("/invite/accept?error=invalid");
  const claim = await sessionPOST("/v1/invitations/claim", new URLSearchParams({ token, return_to: "/invite/accept" }));
  if (claim?.status !== 200) redirect(`/invite/accept?error=${claim?.status === 401 ? "session" : "invalid"}`);
  const { handoff_token } = await claim.json() as { handoff_token: string };
  const result = await sessionPOST("/v1/invitations/accept", new URLSearchParams({ handoff_token }));
  if (result?.status === 401) redirect("/invite/accept?error=session");
  if (!result?.ok) redirect("/invite/accept?error=invalid");
  const organization = await result.json() as { id: string };
  (await cookies()).delete(inviteCookie);
  redirect(`/o/${encodeURIComponent(organization.id)}/tasks`);
}

export default async function AcceptInvitation({ searchParams }: { searchParams: Promise<{ token?: string; error?: string }> }) {
  const query = await searchParams;
  const token = query.token || (await cookies()).get(inviteCookie)?.value;
  const session = token ? await sessionGET("/v1/session") : null;
  return <section className="auth-page"><div className="auth-card"><p className="eyebrow">คำเชิญเข้าร่วมองค์กร</p><h1>เข้าร่วมทีมใน PlaiFlow</h1>
    {query.error && <p className="form-error" role="alert">{query.error === "session" ? "เซสชันหมดอายุ กรุณาเข้าสู่ระบบอีกครั้ง" : "คำเชิญใช้ไม่ได้แล้วหรือหมดอายุ กรุณาขอลิงก์ใหม่จากผู้ดูแล"}</p>}
    {!token ? <><p className="intro">เปิดลิงก์คำเชิญที่ได้รับจากเจ้าของหรือผู้ดูแลองค์กร</p><Link className="button inline-button" href="/organizations">กลับไปเลือกองค์กร</Link></> : session?.ok ? <><p className="intro">คุณกำลังจะรับสิทธิ์สมาชิกขององค์กรที่ส่งคำเชิญนี้</p><form action={accept.bind(null, token)}><button className="button" type="submit">ยืนยันเข้าร่วมองค์กร</button></form></> : <><p className="intro">เข้าสู่ระบบก่อนรับคำเชิญ โดยเลือกบัญชีที่จะใช้ทำงานในองค์กร</p><div className="card-actions"><form action={signIn.bind(null, token, "line")}><button className="button" type="submit">เข้าสู่ระบบด้วย LINE</button></form><form action={signIn.bind(null, token, "google")}><button className="secondary-button" type="submit">เข้าสู่ระบบด้วย Google</button></form></div></>}
  </div></section>;
}
