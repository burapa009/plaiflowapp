"use client";

import { useActionState, useState } from "react";

export type InviteState = { path?: string; error?: string };

export function InviteForm({ action }: { action: (state: InviteState, formData: FormData) => Promise<InviteState> }) {
  const [state, formAction, pending] = useActionState(action, {});
  const [copiedPath, setCopiedPath] = useState("");
  const link = state.path ?? "";
  return <div><form action={formAction}><button className="button" type="submit" disabled={pending}>{pending ? "กำลังสร้างคำเชิญ…" : "สร้างลิงก์เชิญสมาชิก"}</button></form>
    {state.error && <p className="form-error" role="alert">{state.error}</p>}
    {state.path && <div className="invite-result" role="status"><p>ลิงก์นี้ใช้ได้ 24 ชั่วโมง ส่งให้คนที่ต้องการเชิญเป็นสมาชิก</p><input aria-label="ลิงก์เชิญ" readOnly value={link} onFocus={(event) => event.currentTarget.select()} /><button className="secondary-button" type="button" onClick={async () => { try { await navigator.clipboard.writeText(link); setCopiedPath(state.path ?? ""); } catch { document.querySelector<HTMLInputElement>(".invite-result input")?.focus(); } }}>{copiedPath === state.path ? "คัดลอกแล้ว" : "คัดลอกลิงก์เชิญ"}</button><p className="field-help">ลิงก์แสดงครั้งเดียว โปรดคัดลอกก่อนออกจากหน้านี้</p></div>}
  </div>;
}

export function ConfirmButton({ children, message, className = "danger-button" }: { children: React.ReactNode; message: string; className?: string }) {
  return <button className={className} type="submit" onClick={(event) => { if (!window.confirm(message)) event.preventDefault(); }}>{children}</button>;
}
