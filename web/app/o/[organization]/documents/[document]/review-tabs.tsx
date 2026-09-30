"use client";

import { useId, useRef, useState, type KeyboardEvent, type ReactNode } from "react";

export default function ReviewTabs({ documentPanel, amountsPanel }: { documentPanel: ReactNode; amountsPanel: ReactNode }) {
  const [active, setActive] = useState(0);
  const id = useId();
  const tabs = useRef<(HTMLButtonElement | null)[]>([]);
  const panels = [documentPanel, amountsPanel];
  const labels = ["ข้อมูลรายจ่าย", "รายการและสรุปค่าใช้จ่าย"];

  function onKeyDown(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    let next = index;
    if (event.key === "ArrowRight") next = (index + 1) % tabs.current.length;
    else if (event.key === "ArrowLeft") next = (index + tabs.current.length - 1) % tabs.current.length;
    else if (event.key === "Home") next = 0;
    else if (event.key === "End") next = tabs.current.length - 1;
    else return;
    event.preventDefault();
    setActive(next);
    tabs.current[next]?.focus();
  }

  return <div className="ocr-review-tabs">
    <div role="tablist" aria-label="ส่วนตรวจข้อมูลเอกสาร">
      {labels.map((label, index) => <button
        key={label}
        ref={(button) => { tabs.current[index] = button; }}
        id={`${id}-tab-${index}`}
        type="button"
        role="tab"
        aria-controls={`${id}-panel-${index}`}
        aria-selected={active === index}
        tabIndex={active === index ? 0 : -1}
        className={`ocr-review-tab${active === index ? " is-active" : ""}`}
        onClick={() => setActive(index)}
        onKeyDown={(event) => onKeyDown(event, index)}
      >{label}</button>)}
    </div>
    {panels.map((panel, index) => <div
      key={index}
      id={`${id}-panel-${index}`}
      role="tabpanel"
      aria-labelledby={`${id}-tab-${index}`}
      tabIndex={0}
      hidden={active !== index}
      className={`ocr-review-tab-panel${active === index ? " is-active" : ""}`}
    >{panel}</div>)}
  </div>;
}
