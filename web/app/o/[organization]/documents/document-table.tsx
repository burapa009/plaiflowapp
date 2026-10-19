import Link from "next/link";
import { DocumentIcon, DocumentStatus } from "./document-visuals";

export type InboxDocument = {
  id: string;
  filename: string;
  mime: string;
  size: number;
  status: string;
  accepted_at: string;
  source_channel?: string;
};

export function DocumentTable({
  documents,
  organization,
  canManage,
  documentHref,
  changeStatus,
}: {
  documents: InboxDocument[];
  organization: string;
  canManage: boolean;
  documentHref: (id: string) => string;
  changeStatus: (organization: string, document: string, action: "trash" | "restore") => Promise<void>;
}) {
  const root = `/o/${encodeURIComponent(organization)}`;

  function renderTypeBadge(mime: string) {
    const isPDF = mime === "application/pdf";
    const isWebP = mime.includes("webp");
    const isJPEG = mime.includes("jpeg") || mime.includes("jpg");
    const isPNG = mime.includes("png");
    const label = isPDF ? "PDF" : isJPEG ? "JPG" : isPNG ? "PNG" : isWebP ? "WEBP" : mime.replace("image/", "").toUpperCase();
    const typeClass = isPDF ? "is-pdf" : isWebP ? "is-webp" : "is-image";
    return <span className={`doc-type-badge ${typeClass}`}>{label}</span>;
  }

  return (
    <div className="inbox-table-scroll" tabIndex={0} role="region" aria-label="ตารางเอกสาร เลื่อนแนวนอนเพื่อดูทั้งหมด">
      <table className="inbox-table">
        <caption>เอกสารที่คุณเข้าถึงได้ · หน้าละไม่เกิน 20 รายการ</caption>
        <thead>
          <tr>
            <th scope="col">ชื่อไฟล์</th>
            <th scope="col">ชนิดไฟล์</th>
            <th scope="col">ช่องทาง</th>
            <th scope="col">วันที่อัปโหลด</th>
            <th scope="col">ขนาด</th>
            <th scope="col">สถานะไฟล์</th>
            <th scope="col">การดำเนินการ</th>
          </tr>
        </thead>
        <tbody>
          {documents.map((doc) => (
            <tr key={doc.id}>
              <th scope="row">
                <Link href={doc.status === "Available" || doc.status === "Archived" ? documentHref(doc.id) : `${root}/documents/${encodeURIComponent(doc.id)}`}>
                  <DocumentIcon kind="file" />
                  {doc.filename}
                </Link>
              </th>
              <td>{renderTypeBadge(doc.mime)}</td>
              <td>{doc.source_channel || "—"}</td>
              <td>{new Intl.DateTimeFormat("th-TH", { dateStyle: "medium", timeZone: "Asia/Bangkok" }).format(new Date(doc.accepted_at))}</td>
              <td className="inbox-number">{(doc.size / 1024 / 1024).toFixed(2)} MiB</td>
              <td><DocumentStatus status={doc.status} /></td>
              <td>
                <div className="inbox-row-actions">
                  <Link className="secondary-button" href={doc.status === "Available" || doc.status === "Archived" ? documentHref(doc.id) : `${root}/documents/${encodeURIComponent(doc.id)}`}>
                    เปิดตรวจ
                  </Link>
                  {doc.status !== "Trash" && (
                    <a aria-label={`ดาวน์โหลด ${doc.filename}`} className="secondary-button" href={`/api${root}/documents/${encodeURIComponent(doc.id)}/original`}>
                      <DocumentIcon kind="download" />
                    </a>
                  )}
                  {canManage && (
                    <form action={changeStatus.bind(null, organization, doc.id, doc.status === "Trash" ? "restore" : "trash")}>
                      <button className="secondary-button" type="submit" aria-label={`${doc.status === "Trash" ? "กู้คืน" : "ย้ายไปถังขยะ"} ${doc.filename}`}>
                        {doc.status === "Trash" ? "กู้คืน" : <DocumentIcon kind="trash" />}
                      </button>
                    </form>
                  )}
                </div>
              </td>
            </tr>
          ))}
          {documents.length === 0 && (
            <tr>
              <td colSpan={7}>
                <div className="empty-state">
                  <h2>ไม่พบเอกสาร</h2>
                  <p>ลองเปลี่ยนตัวกรอง หรืออัปโหลดเอกสารเพื่อเริ่มต้น</p>
                  <a href="#document-upload" className="secondary-button">อัปโหลดเอกสาร</a>
                </div>
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
