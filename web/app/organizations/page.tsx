import { Card } from "@/components/ui/card";
import { getAPIBaseURL } from "@/lib/api-config";
import { cookies, headers } from "next/headers";
import Link from "next/link";
import { redirect } from "next/navigation";

const csrfCookieName = "__Host-plaiflow-csrf";

type Organization = { id: string; name: string; role: "Owner" | "Admin" | "Member" };
type Membership = { organization_id: string; user_id: string; role: Organization["role"] };

async function apiGET(path: string) {
  const cookieHeader = (await cookies()).toString();
  try {
    return await fetch(`${getAPIBaseURL(process.env)}${path}`, {
      headers: { Cookie: cookieHeader },
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
  } catch {
    return null;
  }
}

async function createOrganization(formData: FormData) {
  "use server";

  const name = String(formData.get("name") ?? "").trim();
  if (!name || Array.from(name).length > 160) redirect("/organizations?error=invalid_name");

  const cookieStore = await cookies();
  const csrf = cookieStore.get(csrfCookieName)?.value;
  const origin = (await headers()).get("origin");
  if (!csrf || !origin) redirect("/auth/error?code=session_expired");

  let response: Response;
  try {
    response = await fetch(`${getAPIBaseURL(process.env)}/v1/organizations`, {
      method: "POST",
      headers: {
        Cookie: cookieStore.toString(),
        "Content-Type": "application/x-www-form-urlencoded",
        Origin: origin,
        "X-CSRF-Token": csrf,
      },
      body: new URLSearchParams({ name }),
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
  } catch {
    redirect("/organizations?error=organization_not_created");
  }

  if (response.status === 401) redirect("/auth/error?code=session_expired");
  if (!response.ok) {
    const body = await response.json().catch(() => ({})) as { code?: string };
    const code = body.code === "invalid_name" ? "invalid_name" : "organization_not_created";
    redirect(`/organizations?error=${code}`);
  }
  const organization = await response.json().catch(() => null) as { id?: string } | null;
  if (!organization?.id) redirect("/organizations?error=organization_not_created");
  redirect(`/o/${encodeURIComponent(organization.id)}/tasks`);
}

export default async function Organizations({ searchParams }: { searchParams: Promise<{ created?: string; error?: string; drive?: string }> }) {
  const params = await searchParams;
  const response = await apiGET("/v1/organizations");
  if (response?.status === 401) redirect("/");
  if (!response?.ok) return <SetupError />;

  const { organizations } = await response.json() as { organizations: Organization[] };
  let membership: Membership | null = null;
  if (organizations.length > 0) {
    const detail = await apiGET(`/v1/o/${encodeURIComponent(organizations[0].id)}`);
    if (!detail?.ok) return <SetupError />;
    ({ membership } = await detail.json() as { membership: Membership });
  }

  return (
    <section className="auth-page">
      <div className="auth-card">
        <p className="auth-brand">PlaiFlow</p>
        {organizations.length === 0 ? (
          <>
            <p className="eyebrow">ขั้นตอนสุดท้าย</p>
            <h1>ตั้งชื่อ Organization แรก</h1>
            <p className="intro">ใช้ชื่อที่ทีมของคุณจำได้ง่าย คุณเปลี่ยนรายละเอียดอื่นภายหลังได้</p>
            {params.error && <p className="form-error" role="alert">{params.error === "invalid_name" ? "กรุณาใส่ชื่อไม่เกิน 160 ตัวอักษร" : "ยังสร้าง Organization ไม่ได้ กรุณาลองอีกครั้ง"}</p>}
            <form action={createOrganization} className="setup-form">
              <label htmlFor="organization-name">ชื่อ Organization</label>
              <input id="organization-name" name="name" maxLength={160} required autoComplete="organization" aria-describedby="organization-help" />
              <p id="organization-help" className="field-help">เช่น บริษัท ปลายโฟลว์</p>
              <button className="button" type="submit">สร้าง Organization</button>
            </form>
          </>
        ) : (
          <>
            <p className="eyebrow">พร้อมใช้งาน</p>
            <h1>{organizations[0].name}</h1>
            <p className="intro">Organization และ Owner Membership ถูกสร้างเรียบร้อยแล้ว</p>
            {params.created && <p className="success-message" role="status">สร้างพื้นที่ทำงานสำเร็จ</p>}
            {params.drive && <p className="form-error" role="alert">{params.drive === "access_denied" ? "ยกเลิกการอนุญาต Google Drive แล้ว ยังไม่มีการเชื่อมต่อใหม่" : "เชื่อมต่อ Google Drive ไม่สำเร็จ กรุณาเปิด Connections แล้วลองใหม่"}</p>}
            <Card className="organization-card">
              <span>สิทธิ์ของคุณ</span>
              <strong>{membership?.role}</strong>
            </Card>
            <Link className="button inline-button" href={`/o/${encodeURIComponent(organizations[0].id)}/tasks`}>เปิดพื้นที่งาน</Link>
          </>
        )}
      </div>
    </section>
  );
}

function SetupError() {
  return (
    <section className="auth-page">
      <div className="auth-card" role="alert">
        <p className="eyebrow">เชื่อมต่อไม่สำเร็จ</p>
        <h1>ยังเปิดพื้นที่ทำงานไม่ได้</h1>
        <p className="intro">ตรวจสอบ API หรือฐานข้อมูล แล้วลองอีกครั้ง</p>
        <Link className="button inline-button" href="/organizations">ลองใหม่</Link>
      </div>
    </section>
  );
}
