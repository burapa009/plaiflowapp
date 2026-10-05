"use client";

import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
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
  const [results, setResults] = useState<string[]>([]);
  const lock = useRef(false);
  const stopped = useRef(false);
  const keys = useRef(new WeakMap<File, string>());

  async function upload(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (preview) { setMessage("หน้าตัวอย่างในเครื่อง — ยังไม่ได้ส่งไฟล์"); return; }
    const form = event.currentTarget;
    if (lock.current) return;
    const files = Array.from((form.elements.namedItem("file") as HTMLInputElement).files || []);
    if (!files.length || files.length > 50 || files.some(file => !file.size || file.size > maxBytes || !["application/pdf", "image/jpeg", "image/png"].includes(file.type))) {
      setMessage("เลือก PDF, JPG หรือ PNG สูงสุด 50 ไฟล์ ขนาดไม่เกิน 20 MiB ต่อไฟล์");
      return;
    }
    const csrf = csrfToken();
    if (!csrf) { setMessage("กรุณาเข้าสู่ระบบใหม่ก่อนส่งเอกสาร"); return; }
    lock.current = true;
    stopped.current = false;
    setResults([]);
    setBusy(true);
    setMessage("กำลังตรวจและส่งเอกสาร…");
    let succeeded = 0;
    try {
      for (const [index, file] of files.entries()) {
      if (stopped.current) break;
      setMessage(`กำลังส่งไฟล์ ${index + 1} / ${files.length}`);
      const data = new FormData();
      data.set("file", file);
      const key = keys.current.get(file) || crypto.randomUUID();
      keys.current.set(file, key);
      try {
      const response = await fetch(`/api/o/${encodeURIComponent(organization)}/documents`, {
        method: "POST", credentials: "same-origin",
        signal: AbortSignal.timeout(120_000),
        headers: { "X-CSRF-Token": csrf, "Idempotency-Key": key },
        body: data,
      });
      const result = await response.json() as { code?: string; duplicate?: boolean };
      if (!response.ok) {
        const messages: Record<string, string> = {
          unsupported_file: "ชนิดไฟล์ไม่รองรับ", file_too_large: "ไฟล์เกิน 20 MiB",
          unsafe_file: "ไฟล์ไม่ผ่านการตรวจความปลอดภัย", scanner_unavailable: "ระบบตรวจไฟล์ยังไม่พร้อม กรุณาลองใหม่",
          document_quota_exhausted: "โควตาเอกสารเต็มแล้ว", document_in_trash: "เอกสารนี้อยู่ในถังขยะ ให้ผู้ดูแลกู้คืนก่อน",
        };
        setResults(previous => [...previous, `${file.name}: ${messages[result.code || ""] || "ส่งไม่สำเร็จ กรุณาลองใหม่"}`]);
        if ([401, 403, 429].includes(response.status) || result.code === "document_quota_exhausted") break;
        continue;
      }
      succeeded++;
      setResults(previous => [...previous, `${file.name}: ${result.duplicate ? "มีเอกสารนี้แล้ว" : "รับเอกสารแล้ว"}`]);
      } catch {
        setResults(previous => [...previous, `${file.name}: เชื่อมต่อไม่สำเร็จ ลองส่งใหม่ได้`]);
        break;
      }
      }
      setMessage(`${stopped.current ? "หยุดคิวแล้ว · " : ""}รับเอกสาร ${succeeded} / ${files.length} ไฟล์`);
      if (succeeded === files.length) { form.reset(); setFilename(""); }
      if (succeeded) router.refresh();
    } finally {
      lock.current = false;
      setBusy(false);
    }
  }

  return <form onSubmit={upload} className="work-form document-upload-form">
    <div className="document-dropzone">
      <span className="document-icon-circle"><DocumentIcon kind="upload" /></span>
      <strong>{filename || "ลากไฟล์มาวางที่นี่"}</strong><span>หรือ</span>
      <span className="document-file-picker"><DocumentIcon kind="file" />เลือกไฟล์</span>
      <p id="document-file-help">รองรับไฟล์: PDF, JPG, JPEG และ PNG · สูงสุด 50 ไฟล์ (ขนาดไม่เกิน 20 MiB ต่อไฟล์)</p>
      <input aria-label="เลือกไฟล์เอกสารหรือลากไฟล์มาวาง" aria-describedby="document-file-help" name="file" type="file" multiple accept="application/pdf,image/jpeg,image/png" required disabled={busy} onChange={(event) => setFilename(event.target.files?.length ? `เลือกแล้ว ${event.target.files.length} ไฟล์` : "")} />
    </div>
    <button className="button document-upload-submit" type="submit" disabled={busy}><DocumentIcon kind="upload" />{busy ? "กำลังส่ง…" : "ส่งเอกสาร"}<DocumentIcon kind="arrow" /></button>
    {busy && <button className="secondary-button" type="button" onClick={() => { stopped.current = true; setMessage("จะหยุดหลังไฟล์ที่กำลังส่งเสร็จ"); }}>หยุดคิวหลังไฟล์นี้</button>}
    {results.length > 0 && <ul className="inbox-upload-results" aria-label="ผลการอัปโหลด">{results.map((result, index) => <li key={index}>{result}</li>)}</ul>}
    {message && <p role="status" aria-live="polite">{message}</p>}
  </form>;
}
