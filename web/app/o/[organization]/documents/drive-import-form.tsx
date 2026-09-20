"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

function csrfToken() {
  const cookie = document.cookie.split("; ").find((entry) => entry.startsWith("__Host-plaiflow-csrf="));
  return cookie ? decodeURIComponent(cookie.split("=").slice(1).join("=")) : "";
}

export function DriveImportForm({ organization }: { organization: string }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const fileId = String(data.get("file_id") || "").trim();
    const revision = String(data.get("revision") || "").trim();
    const csrf = csrfToken();
    if (!fileId || !revision || !csrf) {
      setMessage("กรอก File ID, revision และเข้าสู่ระบบใหม่ก่อนนำเข้า");
      return;
    }
    setBusy(true);
    setMessage("กำลังตรวจสิทธิ์และนำเข้าไฟล์จาก Google Drive…");
    try {
      const response = await fetch(`/api/o/${encodeURIComponent(organization)}/documents/drive`, {
        method: "POST", credentials: "same-origin", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
        body: JSON.stringify({ file_id: fileId, revision }),
      });
      const result = await response.json() as { code?: string; duplicate?: boolean };
      if (!response.ok) {
        const errors: Record<string, string> = { drive_reconnect_required: "Google Drive หมดสิทธิ์ กรุณาเชื่อมต่อใหม่", drive_file_changed: "ไฟล์เปลี่ยนแปลงแล้ว กรุณาเลือกไฟล์ใหม่", document_quota_exhausted: "โควตาเอกสารเต็มแล้ว" };
        setMessage(errors[result.code || ""] || "นำเข้าไฟล์จาก Drive ไม่สำเร็จ");
        return;
      }
      setMessage(result.duplicate ? "ไฟล์นี้อยู่ในเอกสารแล้ว" : "นำเข้าไฟล์จาก Drive แล้ว");
      event.currentTarget.reset();
      router.refresh();
    } catch {
      setMessage("เชื่อมต่อ Google Drive ไม่สำเร็จ");
    } finally {
      setBusy(false);
    }
  }
  return <form onSubmit={submit} className="work-form"><div className="field"><label htmlFor="drive-file-id">Google Drive File ID</label><input id="drive-file-id" name="file_id" required disabled={busy} /><p className="field-help">ใช้ค่าที่ได้จาก Google Picker</p></div><div className="field"><label htmlFor="drive-revision">Revision</label><input id="drive-revision" name="revision" required disabled={busy} /></div><button className="secondary-button" type="submit" disabled={busy}>{busy ? "กำลังนำเข้า…" : "นำเข้า Drive"}</button>{message && <p role="status" aria-live="polite">{message}</p>}</form>;
}
