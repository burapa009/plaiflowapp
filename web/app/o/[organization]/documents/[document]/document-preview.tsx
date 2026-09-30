"use client";

import Image from "next/image";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

const maxBytes = 20 * 1024 * 1024;
type Attachment = { id: string; filename: string; mime: string };

function documentCookie() {
  const cookie = window.document.cookie.split("; ").find((entry) => entry.startsWith("__Host-plaiflow-csrf="));
  return cookie ? decodeURIComponent(cookie.split("=").slice(1).join("=")) : "";
}

const ocrLabels: Record<string, string> = {
  NotScheduled: "ยังไม่เริ่ม OCR", Queued: "รอ OCR", Running: "กำลังอ่านเอกสาร",
  Completed: "OCR เสร็จแล้ว", Failed: "OCR ไม่สำเร็จ", Cancelled: "ยกเลิก OCR",
};

export default function DocumentPreview({ src, filename, mime, documentType = "", ocrStatus = "", organization, document, attachments = [] }: {
  src: string; filename: string; mime: string; documentType?: string; ocrStatus?: string;
  organization?: string; document?: string; attachments?: Attachment[];
}) {
  const router = useRouter();
  const canvas = useRef<HTMLDivElement>(null);
  const picker = useRef<HTMLInputElement>(null);
  const [size, setSize] = useState({ width: 0, height: 0 });
  const [ratio, setRatio] = useState(25 / 33);
  const [zoom, setZoom] = useState(100);
  const [rotation, setRotation] = useState(0);
  const [showFiles, setShowFiles] = useState(true);
  const [load, setLoad] = useState<"loading" | "ready" | "error">("loading");
  const [attempt, setAttempt] = useState(0);
  const [selectedId, setSelectedId] = useState(document || "original");
  const [uploading, setUploading] = useState(false);
  const [uploadMessage, setUploadMessage] = useState("");
  const files = [{ id: document || "original", filename, mime, src }, ...attachments.map((item) => ({ ...item, src: `/api/o/${encodeURIComponent(organization || "")}/documents/${encodeURIComponent(item.id)}/original?preview=1` }))];
  const selected = files.find((item) => item.id === selectedId) || files[0];
  const currentSrc = selected.src;
  const image = selected.mime.startsWith("image/");
  const pdf = selected.mime === "application/pdf";

  async function attach(file: File) {
    if (!organization || !document) return;
    if (!file.size || file.size > maxBytes) { setUploadMessage("เลือกไฟล์ PDF, JPG หรือ PNG ขนาดไม่เกิน 20 MiB"); return; }
    const cookie = documentCookie();
    if (!cookie) { setUploadMessage("กรุณาเข้าสู่ระบบใหม่ก่อนแนบเอกสาร"); return; }
    setUploading(true);
    setUploadMessage("กำลังตรวจและแนบเอกสาร…");
    try {
      const body = new FormData();
      body.append("file", file);
      const response = await fetch(`/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}/attachments`, {
        method: "POST", credentials: "same-origin", headers: { "X-CSRF-Token": cookie, "Idempotency-Key": crypto.randomUUID() }, body,
      });
      const result = await response.json() as { code?: string };
      if (!response.ok) {
        const errors: Record<string, string> = {
          unsupported_file: "ชนิดไฟล์ไม่รองรับ", file_too_large: "ไฟล์เกิน 20 MiB", unsafe_file: "ไฟล์ไม่ผ่านการตรวจความปลอดภัย",
          scanner_unavailable: "ระบบตรวจไฟล์ยังไม่พร้อม กรุณาลองใหม่", document_quota_exhausted: "โควตาเอกสารเต็มแล้ว",
          document_in_trash: "ไฟล์นี้อยู่ในถังขยะ ให้ผู้ดูแลกู้คืนก่อน", document_attachment_conflict: "ไฟล์นี้เป็นเอกสารหลักอยู่แล้ว",
          not_found: "ไม่พบเอกสารหลักหรือไม่มีสิทธิ์แนบไฟล์",
        };
        setUploadMessage(errors[result.code || ""] || "แนบเอกสารไม่สำเร็จ กรุณาลองใหม่");
        return;
      }
      setUploadMessage("แนบเอกสารแล้ว");
      router.refresh();
    } catch {
      setUploadMessage("เชื่อมต่อไม่สำเร็จ กรุณาลองใหม่");
    } finally {
      setUploading(false);
      if (picker.current) picker.current.value = "";
    }
  }

  useEffect(() => {
    if (!canvas.current) return;
    const observer = new ResizeObserver(([entry]) => setSize({ width: entry.contentRect.width, height: entry.contentRect.height }));
    observer.observe(canvas.current);
    return () => observer.disconnect();
  }, []);

  const sideways = rotation % 180 !== 0;
  const availableWidth = Math.max(size.width - 32, 1);
  const availableHeight = Math.max(size.height - 32, 1);
  const fitWidth = sideways ? Math.min(availableHeight, availableWidth * ratio) : Math.min(availableWidth, availableHeight * ratio);
  const sheetWidth = fitWidth * zoom / 100;
  const sheetHeight = sheetWidth / ratio;
  const stageWidth = Math.max(size.width, sideways ? sheetHeight : sheetWidth);
  const stageHeight = Math.max(size.height, sideways ? sheetWidth : sheetHeight);

  return <aside className="ocr-review-viewer" aria-label="เอกสารต้นฉบับ">
    <div className="ocr-review-viewer-head">
      <div className="ocr-review-viewer-type"><span>ประเภทเอกสาร</span><span className="ocr-review-ocr-state">{selected.id !== files[0].id ? "เอกสารแนบ" : ocrStatus === "Completed" ? "Plaiflow อ่านแล้ว" : ocrLabels[ocrStatus] ?? "สถานะ OCR ไม่พร้อม"}</span><select aria-label="ประเภทเอกสาร" value={selected.id === files[0].id ? documentType || "ยังระบุไม่ได้" : "หลักฐานประกอบ"} disabled title="ประเภทเอกสารจาก OCR ยังแก้ไขในหน้านี้ไม่ได้"><option>{selected.id === files[0].id ? documentType || "ยังระบุไม่ได้" : "หลักฐานประกอบ"}</option></select></div>
      <div className="ocr-review-viewer-tags"><span>แท็ก</span><select aria-label="เลือกแท็ก" disabled title="ยังไม่รองรับแท็กเอกสาร"><option>เลือกแท็ก</option></select></div>
      <span className="ocr-review-file-count">ไฟล์ปัจจุบัน: {files.findIndex((item) => item.id === selected.id) + 1} / {files.length}</span>
    </div>
    <div ref={canvas} className="ocr-review-canvas" aria-busy={!!currentSrc && (image || pdf) && load === "loading"}>
      {!currentSrc ? <p className="ocr-review-viewer-message">ยังไม่มีเอกสารต้นฉบับ</p> : !image && !pdf ? <p className="ocr-review-viewer-message">ไม่รองรับตัวอย่างไฟล์ชนิดนี้ <a href={currentSrc} target="_blank" rel="noopener noreferrer">เปิดต้นฉบับ ↗</a></p> : <>
        {load === "loading" && <div className="ocr-review-viewer-message ocr-review-loading" role="status">กำลังโหลดเอกสาร…</div>}
        {load === "error" && <div className="ocr-review-viewer-message" role="alert"><p>โหลดเอกสารไม่สำเร็จ</p><button type="button" className="secondary-button" onClick={() => { setLoad("loading"); setAttempt(value => value + 1); }}>ลองอีกครั้ง</button> <a href={currentSrc} target="_blank" rel="noopener noreferrer">เปิดต้นฉบับ ↗</a></div>}
        {pdf ? <iframe key={`${selected.id}-${attempt}`} className="ocr-review-pdf" title={`เอกสารต้นฉบับ ${selected.filename}`} src={currentSrc} onLoad={() => setLoad("ready")} onError={() => setLoad("error")} /> : size.width > 0 && <div className="ocr-review-stage" style={{ width: stageWidth, height: stageHeight }}>
          <div className="ocr-review-sheet" style={{ width: sheetWidth, height: sheetHeight, transform: `translate(-50%, -50%) rotate(${rotation}deg)` }}>
            {image && <Image key={`${selected.id}-${attempt}`} src={currentSrc} alt={`เอกสารต้นฉบับ ${selected.filename}`} fill sizes="(max-width: 767px) 100vw, 50vw" loading="eager" unoptimized onLoad={(event) => { setRatio(event.currentTarget.naturalWidth / event.currentTarget.naturalHeight); setLoad("ready"); }} onError={() => setLoad("error")} />}
          </div>
        </div>}
      </>}
    </div>
    <div className="ocr-review-viewer-toolbar" role="group" aria-label="เครื่องมือดูเอกสาร">
      {pdf ? <span>ซูม หมุน และเปลี่ยนหน้าด้วยเครื่องมือ PDF</span> : <div className="ocr-review-viewer-controls">
        <button type="button" aria-label="ซูมออก" title="ซูมออก" disabled={!image || !currentSrc || load === "error" || zoom <= 50} onClick={() => setZoom((value) => Math.max(50, value - 25))}>−</button>
        <output aria-label="ระดับซูม">{zoom}%</output>
        <button type="button" aria-label="ซูมเข้า" title="ซูมเข้า" disabled={!image || !currentSrc || load === "error" || zoom >= 200} onClick={() => setZoom((value) => Math.min(200, value + 25))}>+</button>
        <span className="ocr-review-control-divider" aria-hidden="true" />
        <button type="button" aria-label="หมุนซ้าย" title="หมุนซ้าย" disabled={!image || !currentSrc || load === "error"} onClick={() => setRotation((value) => (value + 270) % 360)}>↶</button>
        <button type="button" aria-label="หมุนขวา" title="หมุนขวา" disabled={!image || !currentSrc || load === "error"} onClick={() => setRotation((value) => (value + 90) % 360)}>↷</button>
        <span className="ocr-review-control-divider" aria-hidden="true" />
        <button type="button" title="รีเซ็ตมุมมอง" disabled={!image || !currentSrc || load === "error"} onClick={() => { setZoom(100); setRotation(0); }}>↺ รีเซ็ต</button>
      </div>}
      {currentSrc && <a href={currentSrc} target="_blank" rel="noopener noreferrer">เปิดเต็มหน้า ↗</a>}
    </div>
    <div className="ocr-review-file-list"><div className="ocr-review-file-heading"><div><strong>เอกสารรายจ่าย</strong><small>ประเภทเอกสารทั้งหมด: {documentType || "ยังระบุไม่ได้"}</small></div><button type="button" aria-expanded={showFiles} aria-controls="review-file-thumbnails" onClick={() => setShowFiles((value) => !value)}>{showFiles ? "ซ่อน⌄" : "แสดง⌃"}</button></div>
      {showFiles && <div id="review-file-thumbnails" className="ocr-review-file-thumbnails">{files.filter((item) => item.src).map((item, index) => <button key={item.id} type="button" className="ocr-review-file" aria-pressed={selected.id === item.id} aria-label={`ดู ${item.filename}`} onClick={() => { if (item.id !== selected.id) setLoad("loading"); setSelectedId(item.id); setZoom(100); setRotation(0); }}><span className="ocr-review-thumbnail" aria-hidden="true"><span className="ocr-review-thumbnail-badge">{index === 0 ? ocrStatus === "Completed" ? "Plaiflow อ่านแล้ว" : ocrLabels[ocrStatus] ?? "เอกสาร" : "หลักฐาน"}</span>{item.mime.startsWith("image/") ? <Image src={item.src} alt="" width={80} height={96} loading="lazy" unoptimized /> : "PDF"}<span className="ocr-review-thumbnail-check">✓</span></span><span className="ocr-review-file-label" title={item.filename}>{index === 0 ? documentType || item.filename : item.filename}</span></button>)}<button type="button" className="ocr-review-attach" disabled={!organization || !document || uploading} title={!organization || !document ? "หน้าตัวอย่างไม่รองรับการอัปโหลด" : undefined} onClick={() => picker.current?.click()}><span aria-hidden="true">+</span><span>{uploading ? "กำลังแนบ…" : <>แนบเอกสารรายจ่าย<br />/หลักฐาน</>}</span></button><input ref={picker} className="sr-only" tabIndex={-1} type="file" accept="application/pdf,image/jpeg,image/png" aria-label="เลือกเอกสารรายจ่ายหรือหลักฐานเพื่อแนบ" disabled={uploading} onChange={(event) => { const file = event.currentTarget.files?.[0]; event.currentTarget.value = ""; if (file) void attach(file); }} /></div>}
      {uploadMessage && <p className="ocr-review-upload-message" role="status" aria-live="polite">{uploadMessage}</p>}
    </div>
  </aside>;
}
