"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";

export default function ReviewWorkspace({ viewer, form, support, formLabel = "แก้ไขข้อมูล" }: { viewer: ReactNode; form: ReactNode; support?: ReactNode; formLabel?: string }) {
  const dirty = useRef(false);
  const [mobileView, setMobileView] = useState<"document" | "form">("document");

  useEffect(() => {
    function beforeUnload(event: BeforeUnloadEvent) {
      if (!dirty.current) return;
      event.preventDefault();
      event.returnValue = "";
    }
    function confirmExit(event: MouseEvent) {
      if (!dirty.current || !(event.target instanceof Element) || event.ctrlKey || event.metaKey || event.shiftKey) return;
      const link = event.target.closest<HTMLAnchorElement>("a[href]");
      const destination = link && new URL(link.href);
      if (!link || link.target === "_blank" || (destination?.pathname === location.pathname && destination.search === location.search)) return;
      if (window.confirm("มีข้อมูลที่ยังไม่ได้บันทึก ต้องการออกจากหน้านี้หรือไม่?")) dirty.current = false;
      else { event.preventDefault(); event.stopPropagation(); }
    }
    function confirmOtherAction(event: SubmitEvent) {
      if (!dirty.current || !(event.target instanceof HTMLFormElement) || event.target.matches(".ocr-review-form")) return;
      if (window.confirm("มีข้อมูลที่ยังไม่ได้บันทึก ต้องการดำเนินการต่อและละทิ้งการแก้ไขหรือไม่?")) dirty.current = false;
      else { event.preventDefault(); event.stopPropagation(); }
    }
    const revealForm = () => setMobileView("form");
    const saved = () => { dirty.current = false; };
    const changed = () => { dirty.current = true; };
    document.addEventListener("review-dirty", changed);
    document.addEventListener("review-saved", saved);
    document.addEventListener("review-focus-field", revealForm);
    window.addEventListener("beforeunload", beforeUnload);
    document.addEventListener("click", confirmExit, true);
    document.addEventListener("submit", confirmOtherAction, true);
    return () => { document.removeEventListener("review-dirty", changed); document.removeEventListener("review-saved", saved); document.removeEventListener("review-focus-field", revealForm); window.removeEventListener("beforeunload", beforeUnload); document.removeEventListener("click", confirmExit, true); document.removeEventListener("submit", confirmOtherAction, true); };
  }, []);

  return <div className="ocr-review-shell" data-mobile-view={mobileView} onInputCapture={(event) => { if ((event.target as Element).closest(".ocr-review-form")) dirty.current = true; }} onChangeCapture={(event) => { if ((event.target as Element).closest(".ocr-review-form")) dirty.current = true; }}>
    <div className="ocr-review-mobile-switch" role="group" aria-label="มุมมองตรวจเอกสาร">
      <button type="button" aria-pressed={mobileView === "document"} onClick={() => setMobileView("document")}>ดูเอกสาร</button>
      <button type="button" aria-pressed={mobileView === "form"} onClick={() => setMobileView("form")}>{formLabel}</button>
    </div>
    <div className="ocr-review-workspace">{viewer}<div className="ocr-review-right">{form}{support && <details className="ocr-review-support-drawer"><summary aria-label="หลักฐาน OCR และการจัดหมวด"><span aria-hidden="true">⋯</span><span>หลักฐาน OCR และการจัดหมวด</span></summary>{support}</details>}</div></div>
  </div>;
}
