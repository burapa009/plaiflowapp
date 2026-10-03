import { Card } from "@/components/ui/card";
import { sessionGET, sessionPOST } from "@/lib/session-api";
import { secretaryPilotEnabled } from "@/lib/secretary-pilot";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { BriefingPoll } from "./briefing-poll";

type Item = { source_id: string; title: string; due_on?: string; priority?: string; reason?: string; url?: string; scope?: string; category?: string };
type Briefing = { status: "Queued" | "Ready" | "Failed"; local_day: string; generated_at?: string; possibly_stale: boolean; items: Item[]; remaining_count: number };
type Answer = { intent: string; state: string; explanation?: string; items: Item[] };
type Suggestion = { id: string; template_id: string; title: string; responsible_user_id: string; due_on: string; status: string; task_id?: string };
type Template = { id: string; title: string; responsible_user_id: string; cadence: string; first_due_on: string; active: boolean };
type Preferences = { hidden_categories: string[]; pinned_task_ids: string[] };
type Member = { user_id: string; display_name?: string; role: "Owner" | "Admin" | "Member" };

const intents = [
  ["today", "วันนี้ต้องทำอะไร"], ["why-first", "ทำไมงานนี้มาก่อน"], ["follow-up", "ต้องติดตามใคร"],
  ["meeting-actions", "งานจากประชุมที่ยังเปิด"], ["missing-documents", "เอกสารที่ยังขาด"],
] as const;
const categories = [["Task", "งานทั่วไป"], ["FollowUp", "ติดตามผู้ติดต่อ"], ["MeetingAction", "งานจากประชุม"], ["Routine", "งานประจำ"]] as const;

async function refresh(organization: string) {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/secretary-briefing/refresh`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`/o/${encodeURIComponent(organization)}/secretary${response?.ok ? "?updated=1" : response?.status === 429 ? "?error=too_soon" : "?error=refresh"}`);
}

async function saveTemplate(organization: string, templateID: string, formData: FormData) {
  "use server";
  const body = new URLSearchParams({ title: String(formData.get("title") ?? ""),
    responsible_user_id: String(formData.get("responsible_user_id") ?? ""), cadence: String(formData.get("cadence") ?? ""),
    first_due_on: String(formData.get("first_due_on") ?? ""), active: String(formData.get("active") ?? "true") });
  const path = `/v1/o/${encodeURIComponent(organization)}/routine-templates${templateID ? `/${encodeURIComponent(templateID)}` : ""}`;
  const response = await sessionPOST(path, body);
  if (response?.status === 401) redirect("/");
  redirect(`/o/${encodeURIComponent(organization)}/secretary${response?.ok ? "?updated=1#routines" : "?error=template#routines"}`);
}

async function resolveSuggestion(organization: string, id: string, action: "confirm" | "skip") {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/routine-suggestions/${encodeURIComponent(id)}/${action}`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`/o/${encodeURIComponent(organization)}/secretary${response?.ok ? "?updated=1#routines" : "?error=suggestion#routines"}`);
}

async function saveCategories(organization: string, formData: FormData) {
  "use server";
  const response = await sessionGET(`/v1/o/${encodeURIComponent(organization)}/secretary-preferences`);
  if (response?.status === 401) redirect("/");
  if (!response?.ok) redirect(`/o/${encodeURIComponent(organization)}/secretary?error=preferences`);
  const prefs = await response.json() as Preferences;
  const body = new URLSearchParams();
  for (const category of formData.getAll("hidden_category")) body.append("hidden_category", String(category));
  for (const id of prefs.pinned_task_ids ?? []) body.append("pinned_task_id", id);
  const saved = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/secretary-preferences`, body);
  redirect(`/o/${encodeURIComponent(organization)}/secretary${saved?.ok ? "?updated=1#preferences" : "?error=preferences#preferences"}`);
}

async function togglePin(organization: string, taskID: string) {
  "use server";
  const response = await sessionGET(`/v1/o/${encodeURIComponent(organization)}/secretary-preferences`);
  if (response?.status === 401) redirect("/");
  if (!response?.ok) redirect(`/o/${encodeURIComponent(organization)}/secretary?error=preferences`);
  const prefs = await response.json() as Preferences;
  const pins = new Set(prefs.pinned_task_ids ?? []);
  if (pins.has(taskID)) pins.delete(taskID); else pins.add(taskID);
  const body = new URLSearchParams();
  for (const category of prefs.hidden_categories ?? []) body.append("hidden_category", category);
  for (const id of pins) body.append("pinned_task_id", id);
  const saved = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/secretary-preferences`, body);
  redirect(`/o/${encodeURIComponent(organization)}/secretary${saved?.ok ? "?updated=1" : "?error=preferences"}`);
}

function BriefingItems({ items, manager, organization, prefs }: {
  items: Item[]; manager: boolean; organization: string; prefs: Preferences;
}) {
  const groups = manager ? [
    { label: "งานของคุณ", scope: "personal" },
    { label: "งานทีม", scope: "team" },
  ] : [{ label: "งานของคุณ", scope: "" }];
  return <div className="grid gap-4">{groups.map((group) => {
    const selected = group.scope ? items.filter((item) => item.scope === group.scope) : items;
    if (selected.length === 0) return null;
    return <section key={group.label}><h3>{group.label}</h3><ol className="grid gap-3 pl-5">{selected.map((item) => <li key={`${item.category}-${item.source_id}`} className="rounded-xl border border-line p-4">
      <div className="flex flex-wrap items-start justify-between gap-3"><div><span className="field-help">ลำดับ {items.indexOf(item) + 1}</span> · <Link className="font-semibold text-fg underline" href={item.url || "#"}>{item.title}</Link><p className="field-help">{item.reason}</p></div>
        {item.category !== "Routine" && <form action={togglePin.bind(null, organization, item.source_id)}><button className="text-button" type="submit">{(prefs.pinned_task_ids ?? []).includes(item.source_id) ? "เลิกปักหมุด" : "ปักหมุด"}</button></form>}</div>
    </li>)}</ol></section>;
  })}</div>;
}

export default async function SecretaryPage({ params, searchParams }: {
  params: Promise<{ organization: string }>;
  searchParams: Promise<{ intent?: string; updated?: string; error?: string }>;
}) {
  const { organization } = await params;
  if (!secretaryPilotEnabled(organization)) notFound();
  const query = await searchParams;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const selected = intents.some(([id]) => id === query.intent) ? query.intent : "today";
  const [briefingResponse, suggestionResponse, templateResponse, preferenceResponse, organizationResponse, memberResponse, answerResponse] = await Promise.all([
    sessionGET(`${base}/secretary-briefing`), sessionGET(`${base}/routine-suggestions`),
    sessionGET(`${base}/routine-templates`), sessionGET(`${base}/secretary-preferences`),
    sessionGET(base), sessionGET(`${base}/memberships`), sessionGET(`${base}/secretary-intents/${selected}`),
  ]);
  if ([briefingResponse, suggestionResponse, templateResponse, preferenceResponse, organizationResponse, memberResponse, answerResponse].some((r) => r?.status === 401)) redirect("/");
  if (briefingResponse?.status === 404) notFound();
  if (!briefingResponse?.ok || !suggestionResponse?.ok || !templateResponse?.ok || !preferenceResponse?.ok ||
    !organizationResponse?.ok || !memberResponse?.ok || !answerResponse?.ok) {
    return <section className="error-state" role="alert"><h1>เปิดสรุปงานไม่ได้</h1><p>โปรดลองโหลดอีกครั้ง</p><Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/tasks`}>กลับไปงาน</Link></section>;
  }
  const [briefing, suggestionData, templateData, prefs, orgData, memberData, answer] = await Promise.all([
    briefingResponse.json() as Promise<Briefing>, suggestionResponse.json() as Promise<{ suggestions: Suggestion[] }>,
    templateResponse.json() as Promise<{ templates: Template[] }>, preferenceResponse.json() as Promise<Preferences>,
    organizationResponse.json() as Promise<{ membership: Member }>, memberResponse.json() as Promise<{ memberships: Member[] }>,
    answerResponse.json() as Promise<Answer>,
  ]);
  const manager = orgData.membership.role === "Owner" || orgData.membership.role === "Admin";
  return <section className="workspace-page">
    <div className="work-header"><div><p className="eyebrow">Company Secretary</p><h1>สรุปงานวันนี้</h1><p className="intro">จัดลำดับจากวันครบกำหนดและความสำคัญของงานที่คุณมีสิทธิ์ดู</p></div><Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/tasks`}>งานทั้งหมด</Link></div>
    {query.updated && <p className="success-message" role="status">บันทึกแล้ว</p>}
    {query.error && <p className="form-error" role="alert">{query.error === "too_soon" ? "เพิ่งรีเฟรชสรุปงาน โปรดลองอีกครั้งใน 15 นาที" : "บันทึกไม่สำเร็จ โปรดลองใหม่"}</p>}
    <Card className="work-panel">
      <div className="workspace-section-heading"><h2>งานที่ควรลงมือ</h2><span>{briefing.local_day}</span></div>
      {briefing.status === "Queued" && <BriefingPoll />}
      {briefing.status === "Failed" && <p role="alert">เตรียมสรุปงานไม่สำเร็จ โปรดลองรีเฟรช</p>}
      {briefing.status === "Ready" && <>
        <p className="field-help">สร้างเมื่อ {briefing.generated_at ? new Intl.DateTimeFormat("th-TH", { dateStyle: "medium", timeStyle: "short" }).format(new Date(briefing.generated_at)) : "–"}{briefing.possibly_stale ? " · ข้อมูล Task อาจเปลี่ยนแล้ว" : ""}</p>
        {briefing.items.length === 0 ? <p>ไม่มีงานเกินกำหนดหรือครบกำหนดวันนี้จากข้อมูลที่คุณมีสิทธิ์ดู</p> :
          <BriefingItems items={briefing.items} manager={manager} organization={organization} prefs={prefs} />}
        {briefing.remaining_count > 0 && <p className="field-help">ยังมีอีก {briefing.remaining_count} รายการ · <Link href={`/o/${encodeURIComponent(organization)}/tasks`}>ดูงานทั้งหมด</Link> · <Link href="#routines">ดูข้อเสนองานประจำ</Link></p>}
      </>}
      <form action={refresh.bind(null, organization)}><button className="secondary-button" type="submit">รีเฟรชสรุปงาน</button></form>
    </Card>
    <Card className="work-panel"><h2>ถามเลขานุการ</h2><nav className="flex flex-wrap gap-2" aria-label="คำถามที่เลือกได้">{intents.map(([id, label]) => <Link key={id} className={selected === id ? "button" : "secondary-button"} href={`?intent=${id}`}>{label}</Link>)}</nav>
      <div className="mt-4" role="status">{answer.state === "unconfigured" ? <p>{answer.explanation || "ยังไม่มีแหล่งข้อมูลนี้"}</p> : answer.state === "Queued" ? <p>กำลังเตรียมคำตอบ</p> : answer.items?.length ? <ul className="grid gap-2">{answer.items.map((item) => <li key={item.source_id}><Link className="underline" href={item.url || "#"}>{item.title}</Link>{item.reason && <span className="field-help"> · {item.reason}</span>}</li>)}</ul> : <p>ไม่มีรายการจากแหล่งข้อมูลที่ตั้งค่าแล้ว</p>}</div>
    </Card>
    <Card className="work-panel" id="routines"><h2>ข้อเสนองานประจำ</h2><p className="field-help">ระบบจะไม่สร้าง Task จนกว่าจะมีคนกดยืนยัน</p>
      {suggestionData.suggestions.length === 0 ? <p>ยังไม่มีงานประจำที่รอยืนยัน</p> : <ul className="grid gap-3">{suggestionData.suggestions.map((item) => <li className="rounded-xl border border-line p-4" id={`routine-${item.id}`} key={item.id}><strong>{item.title}</strong><p className="field-help">ถึงรอบ {item.due_on}</p><div className="flex gap-2"><form action={resolveSuggestion.bind(null, organization, item.id, "confirm")}><button className="button" type="submit">ยืนยันสร้าง Task</button></form><form action={resolveSuggestion.bind(null, organization, item.id, "skip")}><button className="secondary-button" type="submit">ข้ามรอบนี้</button></form></div></li>)}</ul>}
      {manager && <><h3 className="mt-5">ตั้งค่างานประจำ</h3><form className="work-form" action={saveTemplate.bind(null, organization, "")}><label>ชื่องาน<input name="title" maxLength={200} required /></label><label>ผู้รับผิดชอบ<select name="responsible_user_id" required>{memberData.memberships.map((member) => <option key={member.user_id} value={member.user_id}>{member.display_name || member.user_id}</option>)}</select></label><label>รอบงาน<select name="cadence"><option value="Weekly">รายสัปดาห์</option><option value="Monthly">รายเดือน</option><option value="Quarterly">รายไตรมาส</option></select></label><label>ครบกำหนดครั้งแรก<input name="first_due_on" type="date" required /></label><button className="button" type="submit">เพิ่มงานประจำ</button></form>
        {templateData.templates.map((template) => <form className="work-form mt-4 border-t border-line pt-4" action={saveTemplate.bind(null, organization, template.id)} key={template.id}><strong>{template.title}</strong><label>ชื่องาน<input name="title" defaultValue={template.title} maxLength={200} required /></label><label>ผู้รับผิดชอบ<select name="responsible_user_id" defaultValue={template.responsible_user_id}>{memberData.memberships.map((member) => <option key={member.user_id} value={member.user_id}>{member.display_name || member.user_id}</option>)}</select></label><label>รอบงาน<select name="cadence" defaultValue={template.cadence}><option value="Weekly">รายสัปดาห์</option><option value="Monthly">รายเดือน</option><option value="Quarterly">รายไตรมาส</option></select></label><label>ครบกำหนดครั้งแรก<input name="first_due_on" type="date" defaultValue={template.first_due_on} required /></label><label>สถานะ<select name="active" defaultValue={template.active ? "true" : "false"}><option value="true">เปิด</option><option value="false">หยุด</option></select></label><button className="secondary-button" type="submit">บันทึกงานประจำ</button></form>)}</>}
    </Card>
    <Card className="work-panel" id="preferences"><h2>หมวดที่แสดง</h2><form className="work-form" action={saveCategories.bind(null, organization)}>{categories.map(([id, label]) => <label className="flex items-center gap-2" key={id}><input type="checkbox" name="hidden_category" value={id} defaultChecked={(prefs.hidden_categories ?? []).includes(id)} />ซ่อน {label}</label>)}<button className="secondary-button" type="submit">บันทึกการแสดงผล</button></form></Card>
    <p className="field-help">สรุปนี้ช่วยจัดลำดับงานจากข้อมูลที่บันทึกไว้ ไม่มีการส่งข้อความหรืออนุมัติรายการให้อัตโนมัติ และไม่ใช่คำแนะนำทางการเงิน</p>
  </section>;
}
