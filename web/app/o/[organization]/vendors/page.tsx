import Link from "next/link";
import { redirect } from "next/navigation";
import { sessionGET, sessionMultipartPOST, sessionPOST } from "@/lib/session-api";

type Vendor = { id: string; display_name: string; contact_code?: string; country: string; tax_id?: string; branch_code?: string };
type PreviewRow = { row: number; status: string; reason?: string; contact: Vendor };
type Preview = { id: string; rows: PreviewRow[]; ready: number; duplicates: number; warnings: number; expires_at: string };
type Membership = { role: "Owner" | "Admin" | "Member" };

async function createVendor(organization: string, formData: FormData) {
  "use server";
  const body = new URLSearchParams();
  for (const key of ["display_name", "contact_code", "country", "tax_id", "branch_code", "customer"]) body.set(key, String(formData.get(key) ?? ""));
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/vendors`, body);
  if (response?.status === 401) redirect("/");
  if (!response?.ok) {
    const error = await response?.json().catch(() => ({})) as { code?: string } | undefined;
    redirect(`/o/${encodeURIComponent(organization)}/vendors?error=${error?.code === "duplicate_vendor" ? "duplicate" : error?.code === "feature_unavailable" ? "plan" : "invalid"}`);
  }
  redirect(`/o/${encodeURIComponent(organization)}/vendors?created=1`);
}

async function previewImport(organization: string, formData: FormData) {
  "use server";
  const file = formData.get("file");
  if (!(file instanceof File) || file.size === 0) redirect(`/o/${encodeURIComponent(organization)}/vendors?error=import`);
  const body = new FormData();
  body.set("file", file);
  const response = await sessionMultipartPOST(`/v1/o/${encodeURIComponent(organization)}/vendor-imports/preview`, body);
  if (response?.status === 401) redirect("/");
  if (!response?.ok) redirect(`/o/${encodeURIComponent(organization)}/vendors?error=${response?.status === 402 ? "plan" : "import"}`);
  const preview = await response.json() as Preview;
  redirect(`/o/${encodeURIComponent(organization)}/vendors?preview=${encodeURIComponent(preview.id)}`);
}

async function commitImport(organization: string, preview: string) {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/vendor-imports/${encodeURIComponent(preview)}/commit`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  if (!response?.ok) redirect(`/o/${encodeURIComponent(organization)}/vendors?error=commit`);
  const result = await response.json() as { created: number; skipped: number };
  redirect(`/o/${encodeURIComponent(organization)}/vendors?imported=${result.created}&skipped=${result.skipped}`);
}

export default async function VendorsPage({ params, searchParams }: {
  params: Promise<{ organization: string }>;
  searchParams: Promise<{ q?: string; cursor?: string; error?: string; created?: string; preview?: string; imported?: string; skipped?: string }>;
}) {
  const { organization } = await params;
  const query = await searchParams;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const search = String(query.q ?? "").trim().slice(0, 240);
  const [vendorResponse, membershipResponse] = await Promise.all([
    sessionGET(`${base}/vendors?limit=50&q=${encodeURIComponent(search)}${query.cursor ? `&cursor=${encodeURIComponent(query.cursor)}` : ""}`),
    sessionGET(base),
  ]);
  if (vendorResponse?.status === 401 || membershipResponse?.status === 401) redirect("/");
  if (!vendorResponse?.ok || !membershipResponse?.ok) return <PageError organization={organization} />;
  const { vendors, next_cursor } = await vendorResponse.json() as { vendors: Vendor[]; next_cursor?: string };
  const { membership } = await membershipResponse.json() as { membership: Membership };
  const canManage = membership.role === "Owner" || membership.role === "Admin";
  let preview: Preview | null = null;
  if (canManage && query.preview) {
    const response = await sessionGET(`${base}/vendor-imports/${encodeURIComponent(query.preview)}`);
    if (response?.ok) preview = await response.json() as Preview;
  }

  return <section>
    <div className="work-header"><div><p className="eyebrow">Master data</p><h1>คู่ค้า</h1><p className="intro">สร้าง ค้นหา นำเข้า และส่งออกข้อมูลผู้ขายของ Organization</p></div><div className="card-actions"><Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/tasks`}>งาน</Link>{canManage && <><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/vendors/export.csv`}>ส่งออก CSV</a><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/vendors/export.xlsx`}>ส่งออก XLSX</a></>}</div></div>
    {query.created && <p className="success-message" role="status">เพิ่มคู่ค้าแล้ว</p>}
    {query.imported !== undefined && <p className="success-message" role="status">นำเข้าสำเร็จ {query.imported} รายการ · ข้าม {query.skipped ?? 0} รายการ</p>}
    {query.error && <p className="form-error" role="alert">{vendorError(query.error)}</p>}
    <form className="search-form" method="get"><label htmlFor="vendor-search">ค้นหาคู่ค้า</label><div className="card-actions"><input id="vendor-search" name="q" defaultValue={search} maxLength={240} placeholder="ชื่อ รหัสคู่ค้า หรือเลขประจำตัวผู้เสียภาษี" /><button className="secondary-button" type="submit">ค้นหา</button></div></form>
    <div className="master-layout">
      {canManage && <aside className="work-stack">
        <section className="card"><h2>เพิ่มคู่ค้า</h2><form action={createVendor.bind(null, organization)} className="work-form">
          <div className="field"><label htmlFor="vendor-name">ชื่อคู่ค้า *</label><input id="vendor-name" name="display_name" maxLength={240} required /></div>
          <div className="form-row"><div className="field"><label htmlFor="vendor-code">รหัสคู่ค้า</label><input id="vendor-code" name="contact_code" maxLength={64} /></div><div className="field"><label htmlFor="vendor-country">ประเทศ</label><input id="vendor-country" name="country" defaultValue="TH" maxLength={2} /></div></div>
          <div className="form-row"><div className="field"><label htmlFor="vendor-tax">เลขประจำตัวผู้เสียภาษี</label><input id="vendor-tax" name="tax_id" inputMode="numeric" maxLength={64} /></div><div className="field"><label htmlFor="vendor-branch">สาขา</label><input id="vendor-branch" name="branch_code" defaultValue="00000" maxLength={32} /></div></div>
          <label className="checkbox-field"><input name="customer" type="checkbox" value="true" /> เป็นลูกค้าด้วย</label><button className="button" type="submit">เพิ่มคู่ค้า</button>
        </form></section>
        <section className="card"><h2>นำเข้าคู่ค้า</h2><p className="field-help">รองรับ CSV หรือ XLSX สูงสุด 10 MB ระบบจะแสดงตัวอย่างก่อนบันทึก</p><form action={previewImport.bind(null, organization)} className="work-form"><div className="field"><label htmlFor="vendor-file">เลือกไฟล์</label><input id="vendor-file" name="file" type="file" accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" required /></div><button className="secondary-button" type="submit">ตรวจสอบไฟล์</button></form></section>
      </aside>}
      <div className="work-stack">
        {preview && <ImportPreview preview={preview} commit={commitImport.bind(null, organization, preview.id)} />}
        <section aria-labelledby="vendor-list"><h2 id="vendor-list">รายการคู่ค้า</h2>{vendors.length === 0 ? <p className="empty-state">ไม่พบคู่ค้า</p> : <div className="vendor-list">{vendors.map((vendor) => <article className="task-card" key={vendor.id}><strong>{vendor.display_name}</strong><span className="task-meta">{vendor.contact_code && <span>{vendor.contact_code}</span>}<span>{vendor.country}</span>{vendor.tax_id && <span>Tax ID {vendor.tax_id}</span>}{vendor.branch_code && <span>สาขา {vendor.branch_code}</span>}</span></article>)}</div>}{next_cursor && <Link className="secondary-button" href={`?q=${encodeURIComponent(search)}&cursor=${encodeURIComponent(next_cursor)}`}>หน้าถัดไป</Link>}</section>
      </div>
    </div>
  </section>;
}

function ImportPreview({ preview, commit }: { preview: Preview; commit: () => Promise<void> }) {
  return <section className="card" aria-labelledby="preview-heading"><h2 id="preview-heading">ตัวอย่างก่อนนำเข้า</h2><p className="task-meta"><span>พร้อมนำเข้า {preview.ready}</span><span>ข้อมูลซ้ำ {preview.duplicates}</span><span>คำเตือน {preview.warnings}</span></p><div className="preview-table" role="region" aria-label="ตัวอย่างข้อมูลนำเข้า" tabIndex={0}><table><thead><tr><th>แถว</th><th>สถานะ</th><th>ชื่อ</th><th>Tax ID</th></tr></thead><tbody>{preview.rows.slice(0, 50).map((row) => <tr key={row.row}><td>{row.row}</td><td>{row.status === "ready" ? "พร้อม" : "ข้อมูลซ้ำ"}</td><td>{row.contact.display_name}</td><td>{row.contact.tax_id || "—"}</td></tr>)}</tbody></table></div><form action={commit}><button className="button" type="submit" disabled={preview.ready === 0}>ยืนยันนำเข้า {preview.ready} รายการ</button></form></section>;
}

function PageError({ organization }: { organization: string }) { return <section className="error-state" role="alert"><h1>ยังเปิดข้อมูลคู่ค้าไม่ได้</h1><Link className="button inline-button" href={`/o/${encodeURIComponent(organization)}/vendors`}>ลองใหม่</Link></section>; }
function vendorError(code: string) { return code === "duplicate" ? "มีคู่ค้าที่ใช้เลขประจำตัวผู้เสียภาษีและสาขานี้แล้ว" : code === "plan" ? "แพ็กเกจปัจจุบันยังไม่รวมฟีเจอร์นี้" : code === "import" ? "ไฟล์ไม่ปลอดภัยหรือรูปแบบไม่ถูกต้อง กรุณาตรวจสอบแล้วลองใหม่" : code === "commit" ? "ยังบันทึกข้อมูลนำเข้าไม่ได้ กรุณาสร้างตัวอย่างใหม่" : "ข้อมูลคู่ค้าไม่ถูกต้อง"; }
