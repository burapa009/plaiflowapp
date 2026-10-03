"use client";

import { useContext, useState } from "react";

import { ReviewErrors } from "./review-form";

export default function ReviewField({ fieldKey, label, initial, proposal, initialDecision, required, attention, money }: {
 fieldKey: string; label: string; initial: string; proposal: string; initialDecision: string; required: boolean; attention: boolean; money: boolean;
}) {
 const [value, setValue] = useState(initial);
 const [decision, setDecision] = useState(initialDecision);
 const errors = useContext(ReviewErrors);
 const error = errors.find(error => error.id === `extraction-${fieldKey}`);
 const id = `extraction-${fieldKey}`;
 return <div className={`ocr-review-field${attention ? " needs-attention" : ""}`}>
  <div className="ocr-review-field-heading"><label htmlFor={id}>{label}{required && <span aria-hidden="true"> *</span>}</label>{attention && <span className="ocr-review-field-warning">ต้องตรวจ</span>}</div>
  <input id={id} name={fieldKey} type={fieldKey === "issue_date" && (!value || /^\d{4}-\d{2}-\d{2}$/.test(value)) ? "date" : "text"} aria-invalid={!!error} aria-required={required} aria-describedby={`${id}-decision-help${error ? ` ${id}-error` : ""}`} maxLength={240} inputMode={money ? "decimal" : undefined} value={value} onChange={event => { setValue(event.target.value); setDecision(""); }} className={money ? "ocr-review-money-input" : ""} />
  <label className="ocr-review-decision" htmlFor={`${id}-decision`}>ผลการตรวจช่องนี้</label>
  <select id={`${id}-decision`} name={`${fieldKey}_decision`} data-review-decision data-field-label={label} data-field-id={id} value={decision} onChange={event => setDecision(event.target.value)}>
   <option value="">ยังไม่ได้ตรวจ</option>
   {value.trim() && <option value={value.trim() === proposal ? "accepted" : "corrected"}>{value.trim() === proposal ? "ตรวจแล้ว · ตรงกับต้นฉบับ" : "ตรวจแล้ว · แก้ไขจากต้นฉบับ"}</option>}
   {!value.trim() && <option value="unknown">ตรวจแล้ว · ไม่ทราบค่า</option>}
  </select>
  {error && <p id={`${id}-error`} className="form-error">{error.message}</p>}
  <p id={`${id}-decision-help`} className="ocr-review-card-intro">{decision ? "ตรวจช่องนี้แล้ว · แก้ค่าเมื่อใดต้องตรวจใหม่" : "ตรวจเทียบต้นฉบับก่อนเลือกผลตรวจ · ไม่ทราบให้เว้นว่าง"}</p>
 </div>;
}
