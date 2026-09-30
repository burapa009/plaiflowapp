"use client";

import { useState } from "react";

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
  if (!items.length) return <p className="ocr-review-empty">OCR ยังแยกรายการสินค้าและบริการไม่ได้ กรุณาตรวจเอกสารต้นฉบับ</p>;
  const item = items[selected];
  return <div className="ocr-review-item-editor">
    <div className="ocr-review-item-list" aria-label="รายการที่ OCR อ่านได้">{items.map((entry, index) => <button key={index} type="button" className={selected === index ? "is-selected" : ""} aria-pressed={selected === index} onClick={() => setSelected(index)}><strong>#{index + 1} {show(entry.description)}</strong><span>ยอดรายการ {show(entry.amount)} {currency}</span></button>)}</div>
    <section className="ocr-review-item-detail" aria-label={`รายละเอียดรายการที่ ${selected + 1}`}><h4>รายการที่ {selected + 1}</h4><dl>{labels.map(([key, label]) => <div key={key}><dt>{label}</dt><dd>{show(item[key])}{["unit_price", "discount", "amount"].includes(key) && item[key] != null ? ` ${currency}` : ""}</dd></div>)}</dl><p>ข้อมูลรายการอ่านจาก OCR และยังแก้ไขรายรายการในหน้านี้ไม่ได้</p></section>
  </div>;
}
