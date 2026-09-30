"use client";

import { useActionState, useEffect, useRef, useState, type ReactNode } from "react";
import { useFormStatus } from "react-dom";
import { formatReviewAmount } from "./review-amount";

export type ReviewState = { error: string };

export function ReviewForm({ action, children }: { action: (state: ReviewState, formData: FormData) => Promise<ReviewState>; children: ReactNode }) {
  const [state, formAction] = useActionState(action, { error: "" });
  const errorRef = useRef<HTMLParagraphElement>(null);
  useEffect(() => { if (state.error) errorRef.current?.focus(); }, [state]);
  return <form action={formAction} className="ocr-review-form" onReset={(event) => event.preventDefault()}>

    {state.error && <p ref={errorRef} tabIndex={-1} className="ocr-review-alert" role="alert">{state.error}</p>}
    <ReviewFields>{children}</ReviewFields>
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
  return <div className="ocr-review-live-total"><span>ยอดชำระ</span><strong aria-live="polite">{formatReviewAmount(value, unit)}</strong></div>;
}
