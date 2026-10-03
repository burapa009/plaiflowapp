"use client";

import { useActionState } from "react";

export type CodeState = { command?: string; expiresAt?: string; error?: string };

export function CodeForm({ action }: { action: (state: CodeState, formData: FormData) => Promise<CodeState> }) {
  const [state, formAction, pending] = useActionState(action, {});
  return <div>
    <form action={formAction}><button className="button" type="submit" disabled={pending}>{pending ? "กำลังสร้างโค้ด…" : state.command ? "สร้างโค้ดใหม่" : "สร้างโค้ดเชื่อมกลุ่ม"}</button></form>
    {state.error && <p className="form-error" role="alert">{state.error}</p>}
    {state.command && <div className="line-code" role="status"><p>ส่งข้อความนี้ในกลุ่ม LINE ภายใน 10 นาที</p><code>{state.command}</code><p className="field-help">ใช้ได้ถึง {state.expiresAt && new Date(state.expiresAt).toLocaleTimeString("th-TH", { timeZone: "Asia/Bangkok", hour: "2-digit", minute: "2-digit" })} น. · ส่งจากบัญชี LINE ที่เชื่อมกับ PlaiFlow ของคุณ โค้ดใช้ได้ครั้งเดียวและโค้ดใหม่จะแทนโค้ดเดิม</p></div>}
  </div>;
}
