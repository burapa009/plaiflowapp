import { Card } from "@/components/ui/card";
import { sessionGET, sessionPOST } from "@/lib/session-api";
import { updateTaskBody } from "@/lib/task-form";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";

type Task = {
  ID: string;
  Title: string;
  Description: string;
  AssigneeUserID: string;
  AssigneeName: string;
  WatcherNames: string[];
  Status: "Open" | "InProgress" | "Done" | "Cancelled";
  Priority: "Normal" | "High" | "Urgent";
  DueOn: string;
  IsOverdue: boolean;
  CreatedAt: string;
  UpdatedAt: string;
};

const labels = {
  status: { Open: "เปิดอยู่", InProgress: "กำลังทำ", Done: "เสร็จแล้ว", Cancelled: "ยกเลิก" },
  priority: { Normal: "ปกติ", High: "สูง", Urgent: "เร่งด่วน" },
};

async function updateTask(organization: string, taskID: string, formData: FormData) {
  "use server";
  const body = updateTaskBody(formData);
  if (!body) redirect(`/o/${encodeURIComponent(organization)}/tasks/${encodeURIComponent(taskID)}?error=invalid_task`);
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/tasks/${encodeURIComponent(taskID)}`, body);
  if (response?.status === 401) redirect("/");
  if (!response?.ok) {
    const error = await response?.json().catch(() => ({})) as { code?: string } | undefined;
    redirect(`/o/${encodeURIComponent(organization)}/tasks/${encodeURIComponent(taskID)}?error=${error?.code === "conflict" ? "conflict" : error?.code === "forbidden" ? "forbidden" : "task_not_updated"}`);
  }
  redirect(`/o/${encodeURIComponent(organization)}/tasks/${encodeURIComponent(taskID)}?updated=1`);
}

export default async function TaskDetailPage({ params, searchParams }: {
  params: Promise<{ organization: string; task: string }>;
  searchParams: Promise<{ created?: string; updated?: string; error?: string }>;
}) {
  const { organization, task: taskID } = await params;
  const query = await searchParams;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const [response, memberResponse] = await Promise.all([
    sessionGET(`${base}/tasks/${encodeURIComponent(taskID)}`),
    sessionGET(`${base}/memberships`),
  ]);
  if (response?.status === 401) redirect("/");
  if (response?.status === 404) notFound();
  if (!response?.ok || !memberResponse?.ok) return <DetailError organization={organization} task={taskID} />;
  const task = await response.json() as Task;
  const { memberships } = await memberResponse.json() as { memberships: Array<{ user_id: string; display_name?: string; role: "Owner" | "Admin" | "Member" }> };
  const save = updateTask.bind(null, organization, taskID);

  return <section className="task-detail"><Link className="back-link" href={`/o/${encodeURIComponent(organization)}/tasks`}>← กลับไปที่รายการงาน</Link>{query.created && <p className="success-message" role="status">สร้างงานเรียบร้อยแล้ว</p>}{query.updated && <p className="success-message" role="status">บันทึกการเปลี่ยนแปลงแล้ว</p>}{query.error && <p className="form-error" role="alert">{detailError(query.error)}</p>}<div className="work-header"><div><p className="eyebrow">Task detail</p><h1>{task.Title}</h1></div><span className={`status-pill${task.IsOverdue ? " status-overdue" : ""}`}>{task.IsOverdue ? "เกินกำหนด" : labels.status[task.Status]}</span></div><Card className="detail-card"><dl className="detail-grid"><div><dt>สถานะ</dt><dd>{labels.status[task.Status]}</dd></div><div><dt>ความสำคัญ</dt><dd>{labels.priority[task.Priority]}</dd></div><div><dt>ผู้รับผิดชอบ</dt><dd>{task.AssigneeName || "ยังไม่มอบหมาย"}</dd></div><div><dt>วันครบกำหนด</dt><dd>{task.DueOn ? thaiDate(task.DueOn) : "ยังไม่กำหนด"}</dd></div></dl><div className="description-block"><h2>รายละเอียด</h2><p>{task.Description || "ยังไม่มีรายละเอียด"}</p></div>{task.WatcherNames.length > 0 && <div className="description-block"><h2>ผู้ติดตาม</h2><p>{task.WatcherNames.join(", ")}</p></div>}<form action={save} className="work-form detail-form"><h2>แก้ไขงาน</h2><div className="field"><label htmlFor="edit-title">ชื่องาน</label><input id="edit-title" name="title" defaultValue={task.Title} maxLength={200} required /></div><div className="field"><label htmlFor="edit-description">รายละเอียด</label><textarea id="edit-description" name="description" defaultValue={task.Description} maxLength={5000} rows={4} /></div><div className="form-row"><div className="field"><label htmlFor="edit-status">สถานะ</label><select id="edit-status" name="status" defaultValue={task.Status}><option value="Open">เปิดอยู่</option><option value="InProgress">กำลังทำ</option><option value="Done">เสร็จแล้ว</option><option value="Cancelled">ยกเลิก</option></select></div><div className="field"><label htmlFor="edit-priority">ความสำคัญ</label><select id="edit-priority" name="priority" defaultValue={task.Priority}><option value="Normal">ปกติ</option><option value="High">สูง</option><option value="Urgent">เร่งด่วน</option></select></div></div><div className="form-row"><div className="field"><label htmlFor="edit-assignee">ผู้รับผิดชอบ</label><select id="edit-assignee" name="assignee_user_id" defaultValue={task.AssigneeUserID}><option value="">ยังไม่มอบหมาย</option>{memberships.map((member) => <option key={member.user_id} value={member.user_id}>{member.display_name || member.user_id} · {member.role}</option>)}</select></div><div className="field"><label htmlFor="edit-due">วันครบกำหนด</label><input id="edit-due" name="due_on" type="date" defaultValue={task.DueOn} /></div></div><button className="button" type="submit">บันทึกการเปลี่ยนแปลง</button></form></Card></section>;
}

function detailError(code: string) {
  if (code === "invalid_task") return "ตรวจสอบข้อมูลที่กรอกแล้วลองอีกครั้ง";
  if (code === "conflict") return "งานถูกเปลี่ยนแปลงแล้ว กรุณาโหลดหน้าใหม่";
  if (code === "forbidden") return "คุณไม่มีสิทธิ์แก้ไขงานนี้";
  return "บันทึกงานไม่สำเร็จ กรุณาลองอีกครั้ง";
}

function DetailError({ organization, task }: { organization: string; task: string }) {
  return <section className="error-state" role="alert"><p className="eyebrow">โหลดไม่สำเร็จ</p><h1>ยังเปิดงานนี้ไม่ได้</h1><p className="intro">ลองโหลดอีกครั้ง หากยังไม่สำเร็จให้กลับไปที่รายการงาน</p><div className="card-actions"><Link className="button inline-button" href={`/o/${encodeURIComponent(organization)}/tasks/${encodeURIComponent(task)}`}>ลองใหม่</Link><Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/tasks`}>กลับรายการงาน</Link></div></section>;
}

function thaiDate(value: string) {
  return new Intl.DateTimeFormat("th-TH", { dateStyle: "long", timeZone: "UTC" }).format(new Date(`${value}T00:00:00Z`));
}
