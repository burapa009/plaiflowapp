"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

const maxBytes = 20 * 1024 * 1024;

function csrfToken() {
  const cookie = document.cookie.split("; ").find((entry) => entry.startsWith("__Host-plaiflow-csrf="));
  return cookie ? decodeURIComponent(cookie.split("=").slice(1).join("=")) : "";
}

export function UploadForm({ organization }: { organization: string }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function upload(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const file = new FormData(form).get("file");
    if (!(file instanceof File) || file.size === 0 || file.size > maxBytes) {
      setMessage("เลือกไฟล์ PDF, JPEG หรือ PNG ขนาดไม่เกิน 20 MiB");
      return;
    }
    const csrf = csrfToken();
    if (!csrf) { setMessage("กรุณาเข้าสู่ระบบใหม่ก่อนส่งเอกสาร"); return; }
    setBusy(true);
    setMessage("กำลังตรวจและส่งเอกสาร…");
    try {
      const response = await fetch(`/api/o/${encodeURIComponent(organization)}/documents`, {
        method: "POST", credentials: "same-origin",
        headers: { "X-CSRF-Token": csrf, "Idempotency-Key": crypto.randomUUID() },
        body: new FormData(form),
      });
      const result = await response.json() as { code?: string; duplicate?: boolean };
      if (!response.ok) {
        const messages: Record<string, string> = {
          unsupported_file: "ชนิดไฟล์ไม่รองรับ", file_too_large: "ไฟล์เกิน 20 MiB",
          unsafe_file: "ไฟล์ไม่ผ่านการตรวจความปลอดภัย", scanner_unavailable: "ระบบตรวจไฟล์ยังไม่พร้อม กรุณาลองใหม่",
          document_quota_exhausted: "โควตาเอกสารเต็มแล้ว", document_in_trash: "เอกสารนี้อยู่ในถังขยะ ให้ผู้ดูแลกู้คืนก่อน",
        };
        setMessage(messages[result.code || ""] || "ส่งเอกสารไม่สำเร็จ กรุณาลองใหม่");
        return;
      }
      setMessage(result.duplicate ? "มีเอกสารนี้แล้ว ระบบเชื่อมรายการเดิมให้" : "รับเอกสารแล้ว");
      form.reset();
      router.refresh();
    } catch {
      setMessage("เชื่อมต่อไม่สำเร็จ กรุณาลองใหม่");
    } finally {
      setBusy(false);
    }
  }

  return <form onSubmit={upload} className="work-form">
    <div className="field"><label htmlFor="document-file">เลือกเอกสาร</label><input id="document-file" name="file" type="file" accept="application/pdf,image/jpeg,image/png" required disabled={busy} /><p className="field-help">PDF, JPEG หรือ PNG ไม่เกิน 20 MiB</p></div>
    <button className="button" type="submit" disabled={busy}>{busy ? "กำลังส่ง…" : "ส่งเอกสาร"}</button>
    {message && <p role="status" aria-live="polite">{message}</p>}
  </form>;
}
