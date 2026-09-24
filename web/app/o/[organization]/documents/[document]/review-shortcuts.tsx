"use client";

import { useEffect } from "react";

export default function ReviewShortcuts({ nextHref, previousHref }: { nextHref: string; previousHref: string }) {
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      const target = event.target as HTMLElement | null;
      if (!event.altKey || event.ctrlKey || event.metaKey || event.repeat) return;
      if (target?.isContentEditable || target?.closest("input,textarea,select,[contenteditable]") || target?.tagName === "BUTTON") return;
      const key = event.key.toLowerCase();
      if (key === "j") {
        const fields = [...document.querySelectorAll<HTMLInputElement>("[id^='extraction-']")];
        if (fields.length) { event.preventDefault(); fields[(fields.indexOf(target as HTMLInputElement)+1) % fields.length].focus(); }
        return;
      }
      if (key === "s") {
        const save = document.getElementById("review-save") as HTMLButtonElement | null;
        if (save) { event.preventDefault(); save.click(); }
      } else if (key === "n" && nextHref) {
        const next = document.querySelector<HTMLAnchorElement>("[data-review-next]");
        if (next) { event.preventDefault(); next.click(); }
      } else if (key === "p" && previousHref) {
        const previous = document.querySelector<HTMLAnchorElement>("[data-review-previous]");
        if (previous) { event.preventDefault(); previous.click(); }
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [nextHref, previousHref]);

  return <p className="mb-3 text-xs text-muted">คีย์ลัด: Alt+J ช่องถัดไป · Alt+S บันทึกฉบับร่าง · Alt+N เอกสารถัดไป · Alt+P เอกสารก่อนหน้า</p>;
}
