import { sessionGET, sessionPOST } from "@/lib/session-api";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

type OCRState = { ocr: { status: string; enabled: boolean; page_count: number; failure_code?: string }; download_url: string };
type OCRResult = { pages: { page_number: number; text: string; lines: { confidence: number }[] }[] };

export default async function OCRPanel({ organization, document, unavailable }: { organization: string; document: string; unavailable: boolean }) {
  const path = `/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}`;
  const response = await sessionGET(`/v1${path}/ocr`);
  if (!response?.ok) return null;
  const state = await response.json() as OCRState;
  const labels: Record<string, string> = { NotScheduled: "ยังไม่ได้เริ่มอ่านเอกสาร", Queued: "รอประมวลผล", Running: "กำลังอ่านข้อความ", Completed: "อ่านเอกสารแล้ว", Failed: "อ่านเอกสารไม่สำเร็จ", Cancelled: "ยกเลิกการประมวลผล" };
  let result: OCRResult | null = null;
  if (state.ocr.status === "Completed" && state.download_url.startsWith("/v1/ocr-results/")) {
    const artifact = await sessionGET(state.download_url);
    if (artifact?.ok) result = await artifact.json() as OCRResult;
  }
  const membershipResponse = await sessionGET(`/v1/o/${encodeURIComponent(organization)}`);
  const membership = membershipResponse?.ok ? await membershipResponse.json() as { role?: string; membership?: { role?: string } } : null;
  const role = membership?.role ?? membership?.membership?.role;
  async function retry() {
    "use server";
    const res = await sessionPOST(`/v1${path}/ocr/retry`, new URLSearchParams());
    if (!res?.ok) redirect(`${path}?ocr=unavailable`);
    revalidatePath(path);
  }
  return <section className="mt-5 rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel" aria-labelledby="ocr-heading">
    <h2 id="ocr-heading">ข้อความจากเอกสาร</h2>
    <p role="status" className="text-muted">{!state.ocr.enabled && state.ocr.status === "NotScheduled" ? "ระบบอ่านข้อความจากเอกสารยังไม่เปิดใช้งาน" : labels[state.ocr.status] ?? "ยังไม่พร้อมใช้งาน"}</p>
    {unavailable && state.ocr.enabled && <p role="alert">ส่งคำขอประมวลผลไม่สำเร็จ กรุณาลองใหม่</p>}
    {state.ocr.status === "Failed" && <p>ลองใหม่หรือตรวจว่าไฟล์เปิดได้และอยู่ในขนาดที่รองรับ</p>}
    {result?.pages.map(page => <article key={page.page_number} className="mt-4 border-t border-line pt-4">
      <h3>หน้า {page.page_number}</h3>
      <p className="whitespace-pre-wrap break-words">{page.text || "ไม่พบข้อความในหน้านี้"}</p>
      <p className="text-sm text-muted">ข้อความจาก OCR อาจคลาดเคลื่อน โปรดตรวจเทียบต้นฉบับ</p>
    </article>)}
    {state.ocr.enabled && (role === "Owner" || role === "Admin") && !["Queued", "Running"].includes(state.ocr.status) &&
      <form action={retry} className="mt-4"><button className="secondary-button" type="submit">ประมวลผลเอกสารอีกครั้ง</button></form>}
  </section>;
}
