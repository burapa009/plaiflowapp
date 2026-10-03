import Image from "next/image";
import { getAPIBaseURL } from "@/lib/api-config";
import { cookies, headers } from "next/headers";
import Link from "next/link";
import { redirect } from "next/navigation";

const csrfCookieName = "__Host-plaiflow-csrf";

type Organization = { id: string; name: string; role: "Owner" | "Admin" | "Member" };
const roleName = { Owner: "เจ้าของ", Admin: "ผู้ดูแล", Member: "สมาชิก" };

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

  return (
    <section className="auth-page">
      <div className="auth-card">
        <Link className="auth-brand app-brand" href="/" aria-label="PlaiFlow หน้าแรก"><Image src="/plaiflow-logo.png" alt="PlaiFlow" width={256} height={128} priority /></Link>
        {organizations.length === 0 ? (
          <>
            <p className="eyebrow">ขั้นตอนสุดท้าย</p>
            <h1>สร้างองค์กรของคุณ</h1>
            <p className="intro">องค์กรคือพื้นที่เก็บงาน เอกสาร สมาชิก และกลุ่ม LINE ของทีมเดียวกัน</p>
            {params.error && <p className="form-error" role="alert">{params.error === "invalid_name" ? "กรุณาใส่ชื่อไม่เกิน 160 ตัวอักษร" : "ยังสร้าง Organization ไม่ได้ กรุณาลองอีกครั้ง"}</p>}
            <form action={createOrganization} className="setup-form">
              <label htmlFor="organization-name">ชื่อองค์กร</label>
              <input id="organization-name" name="name" maxLength={160} required autoComplete="organization" aria-describedby="organization-help" />
              <p id="organization-help" className="field-help">เช่น บริษัท ปลายโฟลว์</p>
              <button className="button" type="submit">สร้างองค์กร</button>
            </form>
          </>
        ) : (
          <>
            <p className="eyebrow">พื้นที่ทำงานของคุณ</p>
            <h1>เลือกองค์กร</h1>
            <p className="intro">เลือกทีมที่ต้องการทำงาน ข้อมูลและสิทธิ์ของแต่ละองค์กรแยกจากกัน</p>
            {params.created && <p className="success-message" role="status">สร้างพื้นที่ทำงานสำเร็จ</p>}
            {params.drive && <p className="form-error" role="alert">{params.drive === "access_denied" ? "ยกเลิกการอนุญาต Google Drive แล้ว ยังไม่มีการเชื่อมต่อใหม่" : "เชื่อมต่อ Google Drive ไม่สำเร็จ กรุณาเปิด Connections แล้วลองใหม่"}</p>}
            <div className="organization-list">{organizations.map((organization) => <article key={organization.id} className="card organization-choice"><div><h2>{organization.name}</h2><span className="status-pill">{roleName[organization.role]}</span></div><Link className="button inline-button" href={`/o/${encodeURIComponent(organization.id)}/tasks`}>เปิดพื้นที่งาน</Link><div className="organization-shortcuts"><Link href={`/o/${encodeURIComponent(organization.id)}/members`}>สมาชิกและสิทธิ์</Link><Link href={`/o/${encodeURIComponent(organization.id)}/line-groups`}>กลุ่ม LINE</Link></div></article>)}</div>
            <details className="organization-create"><summary>สร้างองค์กรใหม่</summary><form action={createOrganization} className="setup-form"><label htmlFor="new-organization-name">ชื่อองค์กร</label><input id="new-organization-name" name="name" maxLength={160} required autoComplete="organization" /><p className="field-help">ใช้ชื่อที่ทีมของคุณจำได้ง่าย</p><button className="button" type="submit">สร้างองค์กร</button></form></details>
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
