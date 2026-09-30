"use client";

import { useId, useState } from "react";

type Value = string | number | boolean | null;
const labels: [string, string][] = [
  ["description", "รายละเอียด"], ["quantity", "จำนวน"], ["unit", "หน่วย"],
  ["unit_price", "ราคาต่อหน่วย"], ["discount", "ส่วนลด"], ["amount", "ยอดรายการ"],
];

function show(value: Value | undefined) {
  return value === null || value === undefined || value === "" ? "—" : String(value);
}

export default function OCRItems({ items, currency }: { items: Record<string, Value>[]; currency: string }) {
  const [selected, setSelected] = useState(0);
  const id = useId();
  if (!items.length) return <p className="ocr-review-empty">OCR ยังแยกรายการสินค้าและบริการไม่ได้ กรุณาตรวจเอกสารต้นฉบับ</p>;
  const active = Math.min(selected, items.length - 1);
  const item = items[active];
  return <div className="ocr-review-item-editor">
    <div className="ocr-review-item-list" aria-label="รายการที่ OCR อ่านได้">{items.map((entry, index) => <button key={index} type="button" className={active === index ? "is-selected" : ""} aria-pressed={active === index} aria-controls={`${id}-detail`} onClick={() => setSelected(index)}><strong>#{index + 1} {show(entry.description)}</strong><span>ยอดรายการ {show(entry.amount)} {currency}</span></button>)}</div>
    <section id={`${id}-detail`} className="ocr-review-item-detail" aria-label={`รายละเอียดรายการที่ ${active + 1}`}><h4>รายการที่ {active + 1}</h4><p>ข้อมูลจาก OCR · ยังไม่รองรับเพิ่ม แก้ไข หรือลบรายการ</p><div className="ocr-review-item-fields">{labels.map(([key, label]) => <div key={key} className={`ocr-review-field${key === "description" ? " is-wide" : ""}`}><label htmlFor={`${id}-${key}`}>{label}{["unit_price", "discount", "amount"].includes(key) && currency ? ` (${currency})` : ""}</label>{key === "description" ? <textarea id={`${id}-${key}`} readOnly value={show(item[key])} rows={3} /> : <input id={`${id}-${key}`} readOnly value={show(item[key])} className={key !== "unit" ? "ocr-review-money-input" : ""} />}</div>)}</div></section>
  </div>;
}
