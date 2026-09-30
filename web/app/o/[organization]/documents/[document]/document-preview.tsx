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
  const image = mime.startsWith("image/");
  const pdf = mime === "application/pdf";

  useEffect(() => {
    if (!canvas.current) return;
    const observer = new ResizeObserver(([entry]) => setSize({ width: entry.contentRect.width, height: entry.contentRect.height }));
    observer.observe(canvas.current);
    return () => observer.disconnect();
  }, []);

  const sideways = rotation % 180 !== 0;
  const availableWidth = Math.max(Math.min(size.width - 32, size.width >= 600 ? 340 : size.width - 32), 1);
  const availableHeight = Math.max(size.height - 32, 1);
  const fitWidth = sideways ? Math.min(availableHeight, availableWidth * ratio) : Math.min(availableWidth, availableHeight * ratio);
  const sheetWidth = fitWidth * zoom / 100;
  const sheetHeight = sheetWidth / ratio;
  const stageWidth = Math.max(size.width, sideways ? sheetHeight : sheetWidth);
  const stageHeight = Math.max(size.height, sideways ? sheetWidth : sheetHeight);

  return <aside className="ocr-review-viewer" aria-label="เอกสารต้นฉบับ">
    <div className="ocr-review-viewer-head"><div><span>ประเภทเอกสาร</span><strong>{documentType || "ยังระบุไม่ได้"}</strong><span className="ocr-review-ocr-state">{ocrLabels[ocrStatus] ?? "สถานะ OCR ไม่พร้อม"}</span></div><span>ไฟล์ปัจจุบัน: {src ? "1 / 1" : "0 / 0"}</span></div>
    <div ref={canvas} className="ocr-review-canvas" aria-busy={!!src && (image || pdf) && load === "loading"}>
      {!src ? <p className="ocr-review-viewer-message">ยังไม่มีเอกสารต้นฉบับ</p> : !image && !pdf ? <p className="ocr-review-viewer-message">ไม่รองรับตัวอย่างไฟล์ชนิดนี้ <a href={src} target="_blank" rel="noopener noreferrer">เปิดต้นฉบับ ↗</a></p> : <>
        {load === "loading" && <p className="ocr-review-viewer-message" role="status">กำลังโหลดเอกสาร…</p>}
        {load === "error" && <p className="ocr-review-viewer-message" role="alert">โหลดเอกสารไม่สำเร็จ <a href={src} target="_blank" rel="noopener noreferrer">เปิดต้นฉบับ ↗</a></p>}
        {pdf ? <iframe className="ocr-review-pdf" title={`เอกสารต้นฉบับ ${filename}`} src={src} onLoad={() => setLoad("ready")} /> : size.width > 0 && <div className="ocr-review-stage" style={{ width: stageWidth, height: stageHeight }}>
          <div className="ocr-review-sheet" style={{ width: sheetWidth, height: sheetHeight, transform: `translate(-50%, -50%) rotate(${rotation}deg)` }}>
            {image && <Image src={src} alt={`เอกสารต้นฉบับ ${filename}`} fill sizes="(max-width: 1023px) 100vw, 50vw" loading="eager" unoptimized onLoad={(event) => { setRatio(event.currentTarget.naturalWidth / event.currentTarget.naturalHeight); setLoad("ready"); }} onError={() => setLoad("error")} />}
          </div>
        </div>}
      </>}
    </div>
    <div className="ocr-review-viewer-toolbar" role="group" aria-label="เครื่องมือดูเอกสาร">
      {pdf ? <span>ซูม หมุน และเปลี่ยนหน้าด้วยเครื่องมือ PDF</span> : <div className="ocr-review-viewer-controls">
        <button type="button" aria-label="ซูมออก" title="ซูมออก" disabled={!image || !src || load === "error" || zoom <= 50} onClick={() => setZoom((value) => Math.max(50, value - 25))}>−</button>
        <output aria-label="ระดับซูม">{zoom}%</output>
        <button type="button" aria-label="ซูมเข้า" title="ซูมเข้า" disabled={!image || !src || load === "error" || zoom >= 200} onClick={() => setZoom((value) => Math.min(200, value + 25))}>+</button>
        <span className="ocr-review-control-divider" />
        <button type="button" aria-label="หมุนซ้าย" title="หมุนซ้าย" disabled={!image || !src || load === "error"} onClick={() => setRotation((value) => (value + 270) % 360)}>↶</button>
        <button type="button" aria-label="หมุนขวา" title="หมุนขวา" disabled={!image || !src || load === "error"} onClick={() => setRotation((value) => (value + 90) % 360)}>↷</button>
        <button type="button" title="พอดีพื้นที่" disabled={!image || !src || load === "error"} onClick={() => { setZoom(100); setRotation(0); }}>พอดีพื้นที่</button>
      </div>}
      {src && <a href={src} target="_blank" rel="noopener noreferrer">เปิดเต็มหน้า ↗</a>}
    </div>
    <div className="ocr-review-file-list"><div><strong>เอกสารรายจ่าย</strong><button type="button" aria-expanded={showFiles} aria-controls="review-file-thumbnails" onClick={() => setShowFiles((value) => !value)}>{showFiles ? "ซ่อน⌃" : "แสดง⌄"}</button></div>
      {showFiles && <div id="review-file-thumbnails" className="ocr-review-file-thumbnails">{src ? <button type="button" className="ocr-review-file" aria-pressed="true" aria-label={`ดู ${filename}`} onClick={() => { setZoom(100); setRotation(0); }}><span className="ocr-review-thumbnail" aria-hidden="true">{image ? <Image src={src} alt="" width={64} height={64} unoptimized /> : "PDF"}</span><span title={filename}>{filename}</span></button> : <p>ยังไม่มีไฟล์แนบ</p>}</div>}
    </div>
  </aside>;
}
