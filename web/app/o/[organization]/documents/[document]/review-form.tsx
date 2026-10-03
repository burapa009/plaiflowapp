"use client";

import { createContext, useActionState, useEffect, useRef, useState, type ReactNode } from "react";
import { useFormStatus } from "react-dom";
import { formatReviewAmount } from "./review-amount";

import { validateReviewField, type FieldError } from "./review-validation";

export const ReviewErrors = createContext<FieldError[]>([]);

export type ReviewState = { error: string };

export function ReviewForm({ action, children, documentType }: { documentType: string; action: (state: ReviewState, formData: FormData) => Promise<ReviewState>; children: ReactNode }) {
  const [state, formAction] = useActionState(action, { error: "" });
  const errorRef = useRef<HTMLDivElement>(null);
  const formRef = useRef<HTMLFormElement>(null);
  const [missing, setMissing] = useState<FieldError[]>([]);
  const [progress, setProgress] = useState({ checked: 0, total: 0 });
  function updateProgress() {
    const fields = Array.from(formRef.current?.querySelectorAll<HTMLSelectElement>("[data-review-decision]") ?? []);
    setProgress({ checked: fields.filter(field => field.value).length, total: fields.length });
    setMissing(current => current.filter(error => { const field = fields.find(field => field.dataset.fieldId === error.id); const input = document.getElementById(error.id) as HTMLInputElement | null; return !field || !input || !field.value || !!validateReviewField(input.name, input.value, input.getAttribute("aria-required") === "true"); }));
  }
  useEffect(() => {
    const fields = Array.from(formRef.current?.querySelectorAll<HTMLSelectElement>("[data-review-decision]") ?? []);
    setProgress({ checked: fields.filter(field => field.value).length, total: fields.length });

  }, []);
  useEffect(() => { if (state.error) errorRef.current?.focus(); }, [state]);
  function focusField(id: string) {
    document.dispatchEvent(new CustomEvent("review-focus-field", { detail: id }));
    requestAnimationFrame(() => { const input = document.getElementById(id); input?.focus(); input?.scrollIntoView({ block: "center" }); });
  }
  return <form ref={formRef} action={formAction} className="ocr-review-form" onReset={(event) => event.preventDefault()} onChange={() => queueMicrotask(updateProgress)} onSubmitCapture={event => {
    const submitter = (event.nativeEvent as SubmitEvent).submitter as HTMLButtonElement | null;
    if (submitter?.value !== "confirm") { setMissing([]); return; }
    const fields = Array.from(event.currentTarget.querySelectorAll<HTMLSelectElement>("[data-review-decision]"));
    const unchecked: FieldError[] = [];
    if (documentType === "unknown") unchecked.push({ id: "document-type-select", label: "ประเภทเอกสาร", message: "เลือกและบันทึกประเภทเอกสารก่อนยืนยัน" });
    for (const field of fields) {
      const input = document.getElementById(field.dataset.fieldId!) as HTMLInputElement;
      const message = validateReviewField(input.name, input.value, input.getAttribute("aria-required") === "true") || (!field.value ? "ยังไม่ได้ตรวจ" : "");
      if (message) unchecked.push({ id: input.id, label: field.dataset.fieldLabel!, message });
    }
    const data = new FormData(event.currentTarget);
    const amounts = ["subtotal", "vat_amount", "total_amount"].map(key => String(data.get(key) ?? ""));
    if (amounts.every(value => /^\d+\.\d{2}$/.test(value))) {
      const cents = amounts.map(value => BigInt(value.replace(".", "")));
      const difference = cents[0] + cents[1] - cents[2];
      if (difference > 2n || difference < -2n) unchecked.push({ id: "extraction-total_amount", label: "ยอดรวมสุทธิ", message: "ยอดก่อนภาษี + ภาษีไม่ตรงกับยอดรวม" });
    }
    setMissing(unchecked);
    if (unchecked.length) { event.preventDefault(); requestAnimationFrame(() => errorRef.current?.focus()); }
  }}>
    {(state.error || missing.length > 0) && <div ref={errorRef} tabIndex={-1} className="ocr-review-alert" role="alert"><p>{state.error || "กรุณาตรวจทุกช่องก่อนยืนยัน"}</p>{missing.length > 0 && <ul>{missing.map(field => <li key={field.id}><button type="button" className="ocr-review-error-link" onClick={() => focusField(field.id)}>{field.label} · {field.message}</button></li>)}</ul>}</div>}
    {progress.total > 0 && <div className="ocr-review-progress" role="status" aria-live="polite">ตรวจแล้ว {progress.checked}/{progress.total} ช่อง<progress value={progress.checked} max={progress.total} aria-label="ช่องที่ตรวจแล้ว" /></div>}
    <ReviewErrors.Provider value={missing}><ReviewFields>{children}</ReviewFields></ReviewErrors.Provider>
  </form>;
}

function ReviewFields({ children }: { children: ReactNode }) {
  const { pending } = useFormStatus();
  return <fieldset className="ocr-review-form-fields" disabled={pending} aria-busy={pending}>{children}</fieldset>;
}

export function ReviewSubmit({ children, intent, className, id, skipValidation = false }: {
  children: ReactNode; intent: string; className: string; id?: string; skipValidation?: boolean;
}) {
  const { pending } = useFormStatus();
  return <button id={id} type="submit" name="intent" value={intent} formNoValidate={skipValidation} disabled={pending} className={className}>{pending ? "กำลังบันทึก…" : children}</button>;
}

export function ReviewTotal({ initial, currency }: { initial: string; currency: string }) {
  const [value, setValue] = useState(initial);
  const [unit, setUnit] = useState(currency);
  useEffect(() => {
    const input = document.getElementById("extraction-total_amount") as HTMLInputElement | null;
    if (!input) return;
    const currencyInput = document.getElementById("extraction-currency") as HTMLInputElement | null;
    const update = () => { setValue(input.value); setUnit(currencyInput?.value ?? currency); };
    currencyInput?.addEventListener("input", update);
    input.addEventListener("input", update);
    return () => { input.removeEventListener("input", update); currencyInput?.removeEventListener("input", update); };
  }, [currency]);
  return <div className="ocr-review-live-total"><span>ยอดรวมเอกสาร</span><strong aria-live="polite">{formatReviewAmount(value, unit)}</strong></div>;
}
