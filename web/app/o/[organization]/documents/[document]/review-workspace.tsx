"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";

export default function ReviewWorkspace({ viewer, form, support }: { viewer: ReactNode; form: ReactNode; support?: ReactNode }) {
  const [view, setView] = useState<"document" | "form">("form");
  const dirty = useRef(false);

  useEffect(() => {
    function beforeUnload(event: BeforeUnloadEvent) {
      if (!dirty.current) return;
      event.preventDefault();
      event.returnValue = "";
    }
    function confirmExit(event: MouseEvent) {
      if (!dirty.current || !(event.target instanceof Element) || event.ctrlKey || event.metaKey || event.shiftKey) return;
      const link = event.target.closest<HTMLAnchorElement>("a[href]");
      const otherAction = event.target.closest(".ocr-review-support form button");
      const destination = link && new URL(link.href);
      if (!otherAction && (!link || link.target === "_blank" || destination?.origin !== location.origin || (destination.pathname === location.pathname && destination.search === location.search))) return;
      if (window.confirm("มีข้อมูลที่ยังไม่ได้บันทึก ต้องการออกจากหน้านี้หรือไม่?")) dirty.current = false;
      else { event.preventDefault(); event.stopPropagation(); }
    }
    window.addEventListener("beforeunload", beforeUnload);
    document.addEventListener("click", confirmExit, true);
    return () => { window.removeEventListener("beforeunload", beforeUnload); document.removeEventListener("click", confirmExit, true); };
  }, []);

  return <div className="ocr-review-shell" data-mobile-view={view} onInputCapture={(event) => { if ((event.target as Element).closest(".ocr-review-form")) dirty.current = true; }} onChangeCapture={(event) => { if ((event.target as Element).closest(".ocr-review-form")) dirty.current = true; }}>
    <div className="ocr-review-mobile-switch" role="group" aria-label="สลับมุมมองเอกสาร"><button type="button" aria-pressed={view === "document"} onClick={() => setView("document")}>ดูเอกสาร</button><button type="button" aria-pressed={view === "form"} onClick={() => setView("form")}>แก้ไขข้อมูล</button></div>
    <div className="ocr-review-workspace">{viewer}<div className="ocr-review-right">{form}{support && <details className="ocr-review-support-drawer"><summary>หลักฐาน OCR และการจัดหมวด</summary>{support}</details>}</div></div>
  </div>;
}
