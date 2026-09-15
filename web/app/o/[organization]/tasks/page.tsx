import { Card } from "@/components/ui/card";
import { sessionGET, sessionPOST } from "@/lib/session-api";
import { taskFormBody } from "@/lib/task-form";
import Link from "next/link";
import { redirect } from "next/navigation";

type Task = {
  ID: string;
  Title: string;
  AssigneeName: string;
  Status: "Open" | "InProgress" | "Done" | "Cancelled";
  Priority: "Normal" | "High" | "Urgent";
  DueOn: string;
};
type Membership = { user_id: string; display_name?: string; role: "Owner" | "Admin" | "Member" };
type Notification = { ID: string; Title: string; DeepLink: string; CreatedAt: string; ReadAt: string | null };

const statusLabel = { Open: "เปิดอยู่", InProgress: "กำลังทำ", Done: "เสร็จแล้ว", Cancelled: "ยกเลิก" };
const priorityLabel = { Normal: "ปกติ", High: "สูง", Urgent: "เร่งด่วน" };

async function createTask(organization: string, formData: FormData) {
  "use server";
  const body = taskFormBody(formData);
  if (!body) redirect(`/o/${encodeURIComponent(organization)}/tasks?error=invalid_task`);
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/tasks`, body);
  if (response?.status === 401) redirect("/");
  if (!response?.ok) {
    const error = await response?.json().catch(() => ({})) as { code?: string } | undefined;
    const code = error?.code === "usage_limit_reached" || error?.code === "feature_unavailable" ? "feature_unavailable" :
      error?.code === "invalid_task" ? "invalid_task" : "task_not_created";
    redirect(`/o/${encodeURIComponent(organization)}/tasks?error=${code}`);
  }
  const task = await response.json() as Task;
  redirect(`/o/${encodeURIComponent(organization)}/tasks/${encodeURIComponent(task.ID)}?created=1`);
}

async function markRead(organization: string, notification: string) {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/notifications/${encodeURIComponent(notification)}/read`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`/o/${encodeURIComponent(organization)}/tasks${response?.ok ? "?read=1" : "?error=notification_not_updated"}`);
}

export default async function TasksPage({ params, searchParams }: {
  params: Promise<{ organization: string }>;
  searchParams: Promise<{ error?: string; read?: string }>;
}) {
  const { organization } = await params;
  const query = await searchParams;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const [taskResponse, memberResponse, notificationResponse] = await Promise.all([
    sessionGET(`${base}/tasks?limit=50`),
    sessionGET(`${base}/memberships`),
    sessionGET(`${base}/notifications?limit=10`),
  ]);
  if ([taskResponse, memberResponse, notificationResponse].some((response) => response?.status === 401)) redirect("/");
  if (!taskResponse?.ok || !memberResponse?.ok || !notificationResponse?.ok) return <WorkError organization={organization} />;

  const { tasks } = await taskResponse.json() as { tasks: Task[] };
  const { memberships } = await memberResponse.json() as { memberships: Membership[] };
  const { notifications } = await notificationResponse.json() as { notifications: Notification[] };
  const create = createTask.bind(null, organization);

  return (
    <section>
      <div className="work-header">
        <div><p className="eyebrow">Organization workspace</p><h1>งานของทีม</h1><p className="intro">สร้าง มอบหมาย และติดตามงานในพื้นที่เดียวกัน</p></div>
        <Link className="secondary-button" href="/organizations">เปลี่ยน Organization</Link>
      </div>
      {query.error && <p className="form-error" role="alert">{errorMessage(query.error)}</p>}
      {query.read && <p className="success-message" role="status">ทำเครื่องหมายว่าอ่านแล้ว</p>}

      <div className="work-layout">
        <Card className="work-panel">
          <h2>สร้างงานใหม่</h2>
          <form action={create} className="work-form">
            <div className="field"><label htmlFor="task-title">ชื่องาน <span aria-hidden="true">*</span></label><input id="task-title" name="title" maxLength={200} required aria-describedby="task-title-help" /><p id="task-title-help" className="field-help">สั้น ชัด และค้นหาได้ง่าย</p></div>
            <div className="field"><label htmlFor="task-description">รายละเอียด</label><textarea id="task-description" name="description" maxLength={5000} rows={4} /></div>
            <div className="form-row">
              <div className="field"><label htmlFor="task-assignee">ผู้รับผิดชอบ</label><select id="task-assignee" name="assignee_user_id"><option value="">ยังไม่มอบหมาย</option>{memberships.map((member) => <option key={member.user_id} value={member.user_id}>{member.display_name || member.user_id} · {member.role}</option>)}</select></div>
              <div className="field"><label htmlFor="task-priority">ความสำคัญ</label><select id="task-priority" name="priority" defaultValue="Normal"><option value="Normal">ปกติ</option><option value="High">สูง</option><option value="Urgent">เร่งด่วน</option></select></div>
            </div>
            <div className="field"><label htmlFor="task-due">วันครบกำหนด</label><input id="task-due" name="due_on" type="date" /></div>
            <button className="button" type="submit">สร้างงาน</button>
          </form>
        </Card>

        <div className="work-stack">
          <section aria-labelledby="task-list-heading"><h2 id="task-list-heading">รายการงาน</h2>{tasks.length === 0 ? <EmptyState /> : <div className="task-list">{tasks.map((task) => <TaskCard key={task.ID} organization={organization} task={task} />)}</div>}</section>
          <section aria-labelledby="inbox-heading"><h2 id="inbox-heading">กล่องแจ้งเตือน</h2>{notifications.length === 0 ? <p className="empty-state">ยังไม่มีการแจ้งเตือนใหม่</p> : <div className="notification-list">{notifications.map((notification) => <Card className="notification-card" key={notification.ID}><div><p className="notification-title">{notification.Title}</p><p className="field-help">{new Intl.DateTimeFormat("th-TH", { dateStyle: "medium", timeStyle: "short" }).format(new Date(notification.CreatedAt))}{notification.ReadAt ? " · อ่านแล้ว" : " · ยังไม่อ่าน"}</p></div><div className="card-actions"><Link className="secondary-button" href={notification.DeepLink}>เปิดงาน</Link>{!notification.ReadAt && <form action={markRead.bind(null, organization, notification.ID)}><button className="text-button" type="submit">อ่านแล้ว</button></form>}</div></Card>)}</div>}</section>
        </div>
      </div>
    </section>
  );
}

function TaskCard({ organization, task }: { organization: string; task: Task }) {
  return <Link className="task-card" href={`/o/${encodeURIComponent(organization)}/tasks/${encodeURIComponent(task.ID)}`}><span className="task-card-title">{task.Title}</span><span className="task-meta"><span>{statusLabel[task.Status]}</span><span>{priorityLabel[task.Priority]}</span>{task.AssigneeName && <span>{task.AssigneeName}</span>}{task.DueOn && <span>ครบกำหนด {thaiDate(task.DueOn)}</span>}</span></Link>;
}

function EmptyState() {
  return <div className="empty-state"><p>ยังไม่มีงานใน Organization นี้</p><p className="field-help">ใช้แบบฟอร์มเพื่อสร้างงานแรกและมอบหมายให้สมาชิกได้ทันที</p></div>;
}

function WorkError({ organization }: { organization: string }) {
  return <section className="error-state" role="alert"><p className="eyebrow">เชื่อมต่อไม่สำเร็จ</p><h1>ยังเปิดพื้นที่งานไม่ได้</h1><p className="intro">ตรวจสอบ API แล้วลองโหลดหน้านี้อีกครั้ง</p><Link className="button inline-button" href={`/o/${encodeURIComponent(organization)}/tasks`}>ลองใหม่</Link></section>;
}

function errorMessage(code: string) {
  if (code === "invalid_task") return "ตรวจสอบชื่องาน วันครบกำหนด และรายละเอียด แล้วลองอีกครั้ง";
  if (code === "feature_unavailable") return "Organization นี้ยังสร้างงานเพิ่มไม่ได้ แต่ยังเปิดดูงานเดิมได้";
  if (code === "notification_not_updated") return "ยังบันทึกสถานะการแจ้งเตือนไม่ได้ กรุณาลองอีกครั้ง";
  return "ยังสร้างงานไม่ได้ กรุณาลองอีกครั้ง";
}

function thaiDate(value: string) {
  return new Intl.DateTimeFormat("th-TH", { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${value}T00:00:00Z`));
}
