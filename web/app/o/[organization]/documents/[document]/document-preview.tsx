"use client";

import Image from "next/image";
import { useEffect, useRef, useState } from "react";

const ocrLabels: Record<string, string> = {
  NotScheduled: "ยังไม่เริ่ม OCR", Queued: "รอ OCR", Running: "กำลังอ่านเอกสาร",
  Completed: "OCR เสร็จแล้ว", Failed: "OCR ไม่สำเร็จ", Cancelled: "ยกเลิก OCR",
};

export default function DocumentPreview({ src, filename, mime, documentType = "", ocrStatus = "" }: {
  src: string; filename: string; mime: string; documentType?: string; ocrStatus?: string;
}) {
  const canvas = useRef<HTMLDivElement>(null);
  const [size, setSize] = useState({ width: 0, height: 0 });
  const [ratio, setRatio] = useState(25 / 33);
  const [zoom, setZoom] = useState(100);
  const [rotation, setRotation] = useState(0);
  const [showFiles, setShowFiles] = useState(true);
  const [load, setLoad] = useState<"loading" | "ready" | "error">("loading");
  const [attempt, setAttempt] = useState(0);
  const image = mime.startsWith("image/");
  const pdf = mime === "application/pdf";

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
      <div className="ocr-review-viewer-type"><span>ประเภทเอกสาร</span><span className="ocr-review-ocr-state">{ocrStatus === "Completed" ? "Paypers อ่าน" : ocrLabels[ocrStatus] ?? "สถานะ OCR ไม่พร้อม"}</span><select aria-label="ประเภทเอกสาร" value={documentType || "ยังระบุไม่ได้"} disabled title="ประเภทเอกสารจาก OCR ยังแก้ไขในหน้านี้ไม่ได้"><option>{documentType || "ยังระบุไม่ได้"}</option></select></div>
      <div className="ocr-review-viewer-tags"><span>แท็ก</span><select aria-label="เลือกแท็ก" disabled title="ยังไม่รองรับแท็กเอกสาร"><option>เลือกแท็ก</option></select></div>
      <span className="ocr-review-file-count">ไฟล์ปัจจุบัน: {src ? "1 / 1" : "0 / 0"}</span>
    </div>
    <div ref={canvas} className="ocr-review-canvas" aria-busy={!!src && (image || pdf) && load === "loading"}>
      {!src ? <p className="ocr-review-viewer-message">ยังไม่มีเอกสารต้นฉบับ</p> : !image && !pdf ? <p className="ocr-review-viewer-message">ไม่รองรับตัวอย่างไฟล์ชนิดนี้ <a href={src} target="_blank" rel="noopener noreferrer">เปิดต้นฉบับ ↗</a></p> : <>
        {load === "loading" && <div className="ocr-review-viewer-message ocr-review-loading" role="status">กำลังโหลดเอกสาร…</div>}
        {load === "error" && <div className="ocr-review-viewer-message" role="alert"><p>โหลดเอกสารไม่สำเร็จ</p><button type="button" className="secondary-button" onClick={() => { setLoad("loading"); setAttempt(value => value + 1); }}>ลองอีกครั้ง</button> <a href={src} target="_blank" rel="noopener noreferrer">เปิดต้นฉบับ ↗</a></div>}
        {pdf ? <iframe key={attempt} className="ocr-review-pdf" title={`เอกสารต้นฉบับ ${filename}`} src={src} onLoad={() => setLoad("ready")} onError={() => setLoad("error")} /> : size.width > 0 && <div className="ocr-review-stage" style={{ width: stageWidth, height: stageHeight }}>
          <div className="ocr-review-sheet" style={{ width: sheetWidth, height: sheetHeight, transform: `translate(-50%, -50%) rotate(${rotation}deg)` }}>
            {image && <Image key={attempt} src={src} alt={`เอกสารต้นฉบับ ${filename}`} fill sizes="(max-width: 767px) 100vw, 50vw" loading="eager" unoptimized onLoad={(event) => { setRatio(event.currentTarget.naturalWidth / event.currentTarget.naturalHeight); setLoad("ready"); }} onError={() => setLoad("error")} />}
          </div>
        </div>}
      </>}
    </div>
    <div className="ocr-review-viewer-toolbar" role="group" aria-label="เครื่องมือดูเอกสาร">
      {pdf ? <span>ซูม หมุน และเปลี่ยนหน้าด้วยเครื่องมือ PDF</span> : <div className="ocr-review-viewer-controls">
        <button type="button" aria-label="ซูมออก" title="ซูมออก" disabled={!image || !src || load === "error" || zoom <= 50} onClick={() => setZoom((value) => Math.max(50, value - 25))}>−</button>
        <output aria-label="ระดับซูม">{zoom}%</output>
        <button type="button" aria-label="ซูมเข้า" title="ซูมเข้า" disabled={!image || !src || load === "error" || zoom >= 200} onClick={() => setZoom((value) => Math.min(200, value + 25))}>+</button>
        <span className="ocr-review-control-divider" aria-hidden="true" />
        <button type="button" aria-label="หมุนซ้าย" title="หมุนซ้าย" disabled={!image || !src || load === "error"} onClick={() => setRotation((value) => (value + 270) % 360)}>↶</button>
        <button type="button" aria-label="หมุนขวา" title="หมุนขวา" disabled={!image || !src || load === "error"} onClick={() => setRotation((value) => (value + 90) % 360)}>↷</button>
        <span className="ocr-review-control-divider" aria-hidden="true" />
        <button type="button" title="รีเซ็ตมุมมอง" disabled={!image || !src || load === "error"} onClick={() => { setZoom(100); setRotation(0); }}>↺ รีเซ็ต</button>
      </div>}
      {src && <a href={src} target="_blank" rel="noopener noreferrer">เปิดเต็มหน้า ↗</a>}
    </div>
    <div className="ocr-review-file-list"><div className="ocr-review-file-heading"><div><strong>เอกสารรายจ่าย</strong><small>ประเภทเอกสารทั้งหมด: {documentType || "ยังระบุไม่ได้"}</small></div><button type="button" aria-expanded={showFiles} aria-controls="review-file-thumbnails" onClick={() => setShowFiles((value) => !value)}>{showFiles ? "ซ่อน⌄" : "แสดง⌃"}</button></div>
      {showFiles && <div id="review-file-thumbnails" className="ocr-review-file-thumbnails">{src && <button type="button" className="ocr-review-file" aria-pressed="true" aria-label={`ดู ${filename}`} onClick={() => { setZoom(100); setRotation(0); }}><span className="ocr-review-thumbnail" aria-hidden="true"><span className="ocr-review-thumbnail-badge">{ocrStatus === "Completed" ? "Paypers อ่าน" : ocrLabels[ocrStatus] ?? "เอกสาร"}</span>{image ? <Image src={src} alt="" width={80} height={96} loading="eager" unoptimized /> : "PDF"}<span className="ocr-review-thumbnail-check">✓</span></span><span className="ocr-review-file-label" title={filename}>{documentType || filename}</span></button>}<button type="button" className="ocr-review-attach" disabled title="ยังไม่รองรับการแนบเอกสารเพิ่มเติมในหน้านี้"><span aria-hidden="true">+</span><span>แนบเอกสารรายจ่าย<br />/หลักฐาน</span></button></div>}
    </div>
  </aside>;
}
