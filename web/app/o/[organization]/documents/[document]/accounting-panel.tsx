import { sessionGET, sessionPOST } from "@/lib/session-api";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

type Category = { id: string; name: string; archived_at?: string };
type Approval = { id: string; revision: number; category_id: string; category_name: string; approved_at: string };
type Rule = { id: string; vendor_id: string; document_type: string; category_id: string; version: number };
type Suggestion = {
  source_review_id: string;
  source_review_revision: number;
  document_type: string;
  reviewed_values: Record<string, string>;
  suggestion: { candidate: { category_id?: string; suggestion_basis: string; confidence: string; warning_codes: string[] };
    vendor_id?: string; rule_set_version: number; status: string; approved?: Approval };
};

export default async function AccountingPanel({ organization, document, nextHref = "" }: { organization: string; document: string; nextHref?: string }) {
  const path = `/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}`;
  const api = `/v1${path}/accounting`;
  const [suggestionResponse, categoriesResponse, membershipResponse] = await Promise.all([
    sessionGET(api), sessionGET(`/v1/o/${encodeURIComponent(organization)}/accounting/categories`),
    sessionGET(`/v1/o/${encodeURIComponent(organization)}`),
  ]);
  if (!suggestionResponse?.ok || !categoriesResponse?.ok) return null;
  const data = await suggestionResponse.json() as Suggestion;
  const { categories } = await categoriesResponse.json() as { categories: Category[] };
  const rulesResponse = data.suggestion.vendor_id ? await sessionGET(`/v1/o/${encodeURIComponent(organization)}/accounting/rules?vendor_id=${encodeURIComponent(data.suggestion.vendor_id)}`) : null;
  const rules = rulesResponse?.ok ? (await rulesResponse.json() as { rules: Rule[] }).rules ?? [] : [];
  const { membership } = membershipResponse?.ok ? await membershipResponse.json() as { membership: { role: string } } : { membership: { role: "" } };
  const canApprove = membership.role === "Owner" || membership.role === "Admin";

  async function createCategory(formData: FormData) {
    "use server";
    const result = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/accounting/categories`,
      new URLSearchParams({ name: String(formData.get("name") ?? "") }));
    if (!result?.ok) redirect(`${path}?accounting=category-error`);
    revalidatePath(path);
    redirect(`${path}?accounting=category-created`);
  }

  async function approve(formData: FormData) {
    "use server";
    const body = new URLSearchParams();
    for (const key of ["review_id", "review_revision", "rule_set_version", "expected_revision", "category_id", "vendor_id", "unmatched_vendor_reason"])
      body.set(key, String(formData.get(key) ?? ""));
    const result = await sessionPOST(`${api}/approve`, body);
    if (!result?.ok) redirect(`${path}?accounting=${result?.status === 409 ? "changed" : "approval-error"}`);
    revalidatePath(path);
    if (formData.get("approve_next") === "1" && nextHref) redirect(nextHref);
    redirect(`${path}?accounting=approved`);
  }

  async function setCategoryArchived(categoryID: string, action: "archive" | "restore") {
    "use server";
    const result = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/accounting/categories/${encodeURIComponent(categoryID)}/${action}`, new URLSearchParams());
    if (!result?.ok) redirect(`${path}?accounting=category-error`);
    revalidatePath(path);
    redirect(`${path}?accounting=category-updated`);
  }

  async function createRule(formData: FormData) {
    "use server";
    const body = new URLSearchParams({ vendor_id: String(formData.get("vendor_id") ?? ""),
      category_id: String(formData.get("category_id") ?? ""), document_type: String(formData.get("document_type") ?? "") });
    const result = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/accounting/rules`, body);
    if (!result?.ok) redirect(`${path}?accounting=rule-error`);
    revalidatePath(path);
    redirect(`${path}?accounting=rule-created`);
  }

  async function renameCategory(categoryID: string, formData: FormData) {
    "use server";
    const result = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/accounting/categories/${encodeURIComponent(categoryID)}/rename`,
      new URLSearchParams({ name: String(formData.get("name") ?? "") }));
    if (!result?.ok) redirect(`${path}?accounting=category-error`);
    revalidatePath(path);
    redirect(`${path}?accounting=category-updated`);
  }

  async function retireRule(ruleID: string) {
    "use server";
    const result = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/accounting/rules/${encodeURIComponent(ruleID)}/retire`, new URLSearchParams());
    if (!result?.ok) redirect(`${path}?accounting=rule-error`);
    revalidatePath(path);
    redirect(`${path}?accounting=rule-retired`);
  }

  async function assignException() {
    "use server";
    const result = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/tasks`,
      new URLSearchParams({ title: "ตรวจข้อยกเว้นเอกสาร", description: path }));
    if (!result?.ok) redirect(`${path}?accounting=task-error`);
    revalidatePath(path);
    redirect(`${path}?accounting=task-created`);
  }

  const active = categories.filter((category) => !category.archived_at);
  return <section className="mt-5 rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel" aria-labelledby="accounting-heading">
    <h2 id="accounting-heading">ข้อเสนอจัดหมวดค่าใช้จ่าย</h2>
    <p className="mt-2 text-sm text-muted">ใช้เฉพาะข้อมูลที่ยืนยันแล้ว · ไม่ใช่การลงบัญชีหรือส่งเข้าระบบบัญชีโดยอัตโนมัติ</p>
    <dl className="mt-4 grid gap-2 text-sm sm:grid-cols-2">
      <div><dt className="text-muted">ชนิดเอกสารที่ยืนยัน</dt><dd>{data.document_type}</dd></div>
      <div><dt className="text-muted">ยอดรวมที่ยืนยัน</dt><dd>{data.reviewed_values.total_amount || "ไม่ระบุ"}</dd></div>
      <div><dt className="text-muted">ที่มาของข้อเสนอ</dt><dd>{data.suggestion.candidate.suggestion_basis}</dd></div>
      <div><dt className="text-muted">สถานะ</dt><dd>{data.suggestion.status}</dd></div>
      <div><dt className="text-muted">ความมั่นใจ</dt><dd>ยังไม่มีคะแนนที่ผ่านการปรับเทียบ</dd></div>
    </dl>
    {data.suggestion.candidate.warning_codes.length > 0 && <p className="mt-3 rounded-xl bg-amber-50 p-3 text-amber-950" role="status">ต้องตรวจ: {data.suggestion.candidate.warning_codes.join(", ")}</p>}
    {data.suggestion.candidate.warning_codes.length > 0 && <form action={assignException} className="mt-3"><button className="secondary-button" type="submit">สร้างงานตรวจข้อยกเว้น</button></form>}
    {data.suggestion.approved && <div className="mt-3 flex flex-wrap items-center gap-3 rounded-xl bg-emerald-50 p-3 text-emerald-950" role="status"><span>อนุมัติหมวด {data.suggestion.approved.category_name} แล้ว</span>{canApprove && <><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/accounting.csv?document_id=${encodeURIComponent(document)}`}>ส่งออก CSV รายการนี้</a><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/accounting.xlsx?document_id=${encodeURIComponent(document)}`}>ส่งออก XLSX รายการนี้</a></>}</div>}
    {canApprove && <>
      <form action={approve} className="mt-5 grid max-w-xl gap-3">
        <input type="hidden" name="review_id" value={data.source_review_id} />
        <input type="hidden" name="review_revision" value={data.source_review_revision} />
        <input type="hidden" name="rule_set_version" value={data.suggestion.rule_set_version} />
        <input type="hidden" name="expected_revision" value={data.suggestion.approved?.revision ?? 0} />
        <input type="hidden" name="vendor_id" value={data.suggestion.vendor_id ?? ""} />
        <label className="font-semibold" htmlFor="expense-category">หมวดค่าใช้จ่าย</label>
        <select id="expense-category" name="category_id" required defaultValue={data.suggestion.approved?.category_id ?? data.suggestion.candidate.category_id ?? ""} className="min-h-11 rounded-xl border border-line bg-surface px-3">
          <option value="">เลือกหมวดหลังตรวจเอกสาร</option>
          {active.map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}
        </select>
        {!data.suggestion.vendor_id && <label className="grid gap-1">เหตุผลที่ยังไม่ระบุผู้ขาย
          <input name="unmatched_vendor_reason" required maxLength={240} className="min-h-11 rounded-xl border border-line bg-surface px-3" />
        </label>}
        <label className="flex items-start gap-2 text-sm"><input type="checkbox" required className="mt-1" />ฉันตรวจเอกสารและยืนยันหมวดนี้แล้ว</label>
        <button className="button w-fit" type="submit" disabled={active.length === 0}>อนุมัติข้อเสนอ</button>
        {nextHref && <button className="secondary-button w-fit" name="approve_next" value="1" type="submit" disabled={active.length === 0}>อนุมัติและไป{nextHref.includes("/review?") ? "คิว" : "เอกสารถัดไป"}</button>}
      </form>
      <form action={createCategory} className="mt-5 flex max-w-xl flex-wrap items-end gap-3 border-t border-line pt-4">
        <label className="grid flex-1 gap-1">เพิ่มหมวดค่าใช้จ่ายของทีม
          <input name="name" required maxLength={120} className="min-h-11 rounded-xl border border-line bg-surface px-3" />
        </label>
        <button className="secondary-button" type="submit">เพิ่มหมวด</button>
      </form>
      {categories.length > 0 && <ul className="mt-3 grid gap-2 text-sm">{categories.map((category) => <li key={category.id} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-line px-3 py-2">
        {category.archived_at ? <span>{category.name} · เก็บถาวร</span> : <form action={renameCategory.bind(null, category.id)} className="flex flex-wrap items-center gap-2"><label className="sr-only" htmlFor={`category-${category.id}`}>ชื่อหมวด</label><input id={`category-${category.id}`} name="name" defaultValue={category.name} required maxLength={120} className="min-h-10 rounded-xl border border-line bg-surface px-2" /><button className="secondary-button" type="submit">เปลี่ยนชื่อ</button></form>}
        <form action={setCategoryArchived.bind(null, category.id, category.archived_at ? "restore" : "archive")}><button className="secondary-button" type="submit">{category.archived_at ? "คืนค่า" : "เก็บถาวร"}</button></form>
      </li>)}</ul>}
      {data.suggestion.vendor_id && data.suggestion.approved && <form action={createRule} className="mt-5 grid max-w-xl gap-3 border-t border-line pt-4">
        <h3 className="font-semibold">สร้างกฎสำหรับเอกสารในอนาคต</h3>
        <p className="text-sm text-muted">เป็นการอนุมัติกฎแยกจากเอกสารนี้ ไม่อนุมัติเอกสารอื่นอัตโนมัติ</p>
        <input type="hidden" name="vendor_id" value={data.suggestion.vendor_id} />
        <input type="hidden" name="category_id" value={data.suggestion.approved.category_id} />
        <label className="grid gap-1">ขอบเขตกฎ<select name="document_type" className="min-h-11 rounded-xl border border-line bg-surface px-3">
          <option value={data.document_type}>ผู้ขายนี้ + {data.document_type}</option>
          <option value="">ผู้ขายนี้ทุกชนิดเอกสาร (ค่าเริ่มต้น)</option>
        </select></label>
        <button className="secondary-button w-fit" type="submit">อนุมัติกฎแยกต่างหาก</button>
      </form>}
      {rules.length > 0 && <div className="mt-4 border-t border-line pt-4"><h3 className="font-semibold">กฎที่อนุมัติสำหรับผู้ขายรายนี้</h3><ul className="mt-2 grid gap-2 text-sm">{rules.map((rule) => <li key={rule.id} className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-line p-3"><span>{rule.document_type || "ทุกชนิดเอกสาร"} → {categories.find((category) => category.id === rule.category_id)?.name ?? "หมวดเดิม"} · v{rule.version}</span><form action={retireRule.bind(null, rule.id)}><button className="secondary-button" type="submit">ยกเลิกกฎ</button></form></li>)}</ul></div>}
    </>}
  </section>;
}
