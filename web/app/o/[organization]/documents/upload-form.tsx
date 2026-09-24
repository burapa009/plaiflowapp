"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { DocumentIcon } from "./document-visuals";

const maxBytes = 20 * 1024 * 1024;

function csrfToken() {
  const cookie = document.cookie.split("; ").find((entry) => entry.startsWith("__Host-plaiflow-csrf="));
  return cookie ? decodeURIComponent(cookie.split("=").slice(1).join("=")) : "";
}

export function UploadForm({ organization, preview = false }: { organization: string; preview?: boolean }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [filename, setFilename] = useState("");

  async function upload(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (preview) { setMessage("หน้าตัวอย่างในเครื่อง — ยังไม่ได้ส่งไฟล์"); return; }
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
      setFilename("");
      router.refresh();
    } catch {
      setMessage("เชื่อมต่อไม่สำเร็จ กรุณาลองใหม่");
    } finally {
      setBusy(false);
    }
  }

  return <form onSubmit={upload} className="work-form document-upload-form">
    <div className="document-dropzone">
      <span className="document-icon-circle"><DocumentIcon kind="upload" /></span>
      <strong>{filename || "ลากไฟล์มาวางที่นี่"}</strong><span>หรือ</span>
      <span className="document-file-picker"><DocumentIcon kind="file" />เลือกไฟล์</span>
      <p id="document-file-help">รองรับไฟล์: PDF, JPG, JPEG, PNG (ขนาดไม่เกิน 20 MiB ต่อไฟล์)</p>
      <input aria-label="เลือกไฟล์เอกสารหรือลากไฟล์มาวาง" aria-describedby="document-file-help" name="file" type="file" accept="application/pdf,image/jpeg,image/png" required disabled={busy} onChange={(event) => setFilename(event.target.files?.[0]?.name || "")} />
    </div>
    <button className="button document-upload-submit" type="submit" disabled={busy}><DocumentIcon kind="upload" />{busy ? "กำลังส่ง…" : "ส่งเอกสาร"}<DocumentIcon kind="arrow" /></button>
    {message && <p role="status" aria-live="polite">{message}</p>}
  </form>;
}
