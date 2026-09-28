import { Card } from "@/components/ui/card";
import { sessionGET, sessionPOST } from "@/lib/session-api";
import { secretaryPilotEnabled } from "@/lib/secretary-pilot";
import { redirect } from "next/navigation";

type Source = { category: "Task" | "FollowUp" | "MeetingAction"; follow_up_with?: string; meeting_name?: string; meeting_on?: string };

async function saveSource(organization: string, task: string, category: Source["category"], formData: FormData) {
  "use server";
  const body = new URLSearchParams({ category });
  if (category === "FollowUp") body.set("follow_up_with", String(formData.get("follow_up_with") ?? ""));
  if (category === "MeetingAction") {
    body.set("meeting_name", String(formData.get("meeting_name") ?? ""));
    body.set("meeting_on", String(formData.get("meeting_on") ?? ""));
  }
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/tasks/${encodeURIComponent(task)}/secretary-source`, body);
  if (response?.status === 401) redirect("/");
  redirect(`/o/${encodeURIComponent(organization)}/tasks/${encodeURIComponent(task)}${response?.ok ? "?source=saved" : "?source=error"}`);
}

export async function SecretaryTaskSource({ organization, task, creatorUserID, result }: {
  organization: string; task: string; creatorUserID: string; result?: string;
}) {
  if (!secretaryPilotEnabled(organization)) return null;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const [sourceResponse, orgResponse] = await Promise.all([
    sessionGET(`${base}/tasks/${encodeURIComponent(task)}/secretary-source`), sessionGET(base),
  ]);
  if (!sourceResponse?.ok || !orgResponse?.ok) return null;
  const source = await sourceResponse.json() as Source;
  const { membership } = await orgResponse.json() as { membership: { user_id: string; role: string } };
  const canEdit = membership.role === "Owner" || membership.role === "Admin" || membership.user_id === creatorUserID;
  return <Card className="detail-card"><h2>ที่มาของงานในสรุปรายวัน</h2>
    {result === "saved" && <p className="success-message" role="status">บันทึกที่มาแล้ว</p>}
    {result === "error" && <p className="form-error" role="alert">บันทึกที่มาไม่สำเร็จ ตรวจสอบข้อมูลและสิทธิ์อีกครั้ง</p>}
    <p>{source.category === "FollowUp" ? `ติดตามผู้ติดต่อ: ${source.follow_up_with}` :
      source.category === "MeetingAction" ? `งานจากประชุม: ${source.meeting_name} · ${source.meeting_on}` : "งานทั่วไป"}</p>
    {canEdit && <div className="grid gap-4 md:grid-cols-2">
      <form className="work-form" action={saveSource.bind(null, organization, task, "FollowUp")}><h3>ติดตามผู้ติดต่อ</h3><label>ผู้ที่ต้องติดตาม<input name="follow_up_with" defaultValue={source.follow_up_with || ""} maxLength={200} required /></label><button className="secondary-button" type="submit">บันทึกการติดตาม</button></form>
      <form className="work-form" action={saveSource.bind(null, organization, task, "MeetingAction")}><h3>งานจากประชุม</h3><label>ชื่อประชุม<input name="meeting_name" defaultValue={source.meeting_name || ""} maxLength={200} required /></label><label>วันที่ประชุม<input name="meeting_on" type="date" defaultValue={source.meeting_on || ""} required /></label><button className="secondary-button" type="submit">บันทึกที่มาจากประชุม</button></form>
      {source.category !== "Task" && <form action={saveSource.bind(null, organization, task, "Task")}><button className="text-button" type="submit">กลับเป็นงานทั่วไป</button></form>}
    </div>}
  </Card>;
}
