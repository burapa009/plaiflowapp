import type { PlanDefinition } from "@/lib/pricing";

export function PlanArtwork({ name }: { name: PlanDefinition["key"] }) {
  if (name === "AccountingFirm") return null;
  return <svg className="plan-artwork" aria-hidden="true" viewBox="0 0 144 112" fill="none">
    <defs><linearGradient id={`art-${name}`} x1="0" y1="0" x2="1" y2="1"><stop stopColor="#41c9f8" /><stop offset="1" stopColor="#1763e9" /></linearGradient></defs>
    <ellipse cx="76" cy="75" rx="62" ry="36" fill="#dff8ff" />
    {name === "Free" ? <><rect x="48" y="15" width="57" height="77" rx="11" fill="#2777ee" transform="rotate(10 48 15)" /><rect x="37" y="12" width="57" height="77" rx="11" fill="#f8fdff" stroke="#cfe5fa" strokeWidth="2" transform="rotate(10 37 12)" /><path d="m52 41 35 6M50 56l31 6M47 71l23 5" stroke="#6db5f3" strokeWidth="5" strokeLinecap="round" /><path d="m111 12 3 8 8 3-8 3-3 8-3-8-8-3 8-3z" fill="#4ea7f8" /></> : name === "Starter" ? <><circle cx="72" cy="32" r="17" fill="url(#art-Starter)" /><path d="M46 83c0-21 11-31 26-31s26 10 26 31" fill="url(#art-Starter)" /><circle cx="33" cy="49" r="11" fill="#83c5fb" /><path d="M14 84c0-15 7-24 19-24 6 0 11 2 14 6" fill="#83c5fb" /><circle cx="111" cy="49" r="11" fill="#45c7d1" /><path d="M97 66c3-4 8-6 14-6 12 0 19 9 19 24" fill="#45c7d1" /></> : name === "Business" ? <><rect x="24" y="69" width="22" height="28" rx="5" fill="#2484ed" /><rect x="55" y="55" width="22" height="42" rx="5" fill="#27a9d4" /><rect x="86" y="35" width="22" height="62" rx="5" fill="#14b9bf" /><path d="m29 55 29-16 17 4 37-30m0 0-3 19m3-19-19 4" stroke="#2599ec" strokeWidth="7" strokeLinecap="round" strokeLinejoin="round" /></> : <><path d="M60 69 97 20c7-9 17-10 25-10 0 8-1 18-10 25L63 72z" fill="url(#art-Growth)" /><circle cx="98" cy="33" r="9" fill="#d8f9ff" /><path d="m59 68-15 2-15 16 21-6zM75 52l4-15 15-13-5 23z" fill="#58b9ed" /><path d="m51 80-9 15m19-17-4 22" stroke="#3dd1d6" strokeWidth="6" strokeLinecap="round" /></>}
  </svg>;
}
