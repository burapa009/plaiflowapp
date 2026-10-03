"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";

export function PlanCarousel({ children }: { children: ReactNode }) {
  const trackRef = useRef<HTMLDivElement>(null);
  const [edges, setEdges] = useState({ first: true, last: false });

  function updateEdges() {
    const track = trackRef.current;
    if (!track) return;
    setEdges({ first: track.scrollLeft < 2, last: track.scrollLeft >= track.scrollWidth - track.clientWidth - 2 });
  }

  useEffect(() => {
    const track = trackRef.current;
    if (!track) return;
    const observer = new ResizeObserver(updateEdges);
    observer.observe(track);
    updateEdges();
    return () => observer.disconnect();
  }, []);

  function slide(direction: -1 | 1) {
    const track = trackRef.current;
    const card = track?.querySelector<HTMLElement>(".plan-card");
    if (!track || !card) return;
    const gap = Number.parseFloat(getComputedStyle(track).columnGap) || 0;
    track.scrollBy({ left: direction * (card.offsetWidth + gap), behavior: matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth" });
  }

  return <div className="pricing-carousel">
    <div className="pricing-carousel-toolbar"><p>เลื่อนดูแพ็กเกจทั้งหมด</p><div className="pricing-carousel-controls"><button type="button" aria-label="แพ็กเกจก่อนหน้า" disabled={edges.first} onClick={() => slide(-1)}><Chevron direction="left" /></button><button type="button" aria-label="แพ็กเกจถัดไป" disabled={edges.last} onClick={() => slide(1)}><Chevron direction="right" /></button></div></div>
    <div className="pricing-track" ref={trackRef} role="region" aria-label="แพ็กเกจ PlaiFlow" tabIndex={0} onScroll={updateEdges}>{children}</div>
  </div>;
}

function Chevron({ direction }: { direction: "left" | "right" }) {
  return <svg aria-hidden="true" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d={direction === "left" ? "m15 5-7 7 7 7" : "m9 5 7 7-7 7"} /></svg>;
}
