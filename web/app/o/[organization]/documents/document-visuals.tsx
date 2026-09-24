export function DocumentIcon({ kind }: { kind: "file" | "upload" | "download" | "trash" | "arrow" }) {
  const paths = {
    file: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z M14 2v6h6 M8 13h8 M8 17h8",
    upload: "M7 18H5a4 4 0 0 1-.8-7.9A7 7 0 0 1 18 8a5 5 0 0 1 1 9.9h-2 M12 21V11 M8 15l4-4 4 4",
    download: "M12 3v12 M7 10l5 5 5-5 M5 17v4h14v-4",
    trash: "M3 6h18 M9 6V3h6v3 M5 6l1 15h12l1-15 M10 10v7 M14 10v7",
    arrow: "M4 12h16 M14 6l6 6-6 6",
  };
  return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d={paths[kind]} /></svg>;
}

export function DocumentSummary({ filename, size, status }: { filename: string; size: number; status: string }) {
  const extension = filename.split(".").pop()?.toUpperCase() || "FILE";
  return <div className="document-file-summary">
    <div className="document-thumbnail" aria-hidden="true"><DocumentIcon kind="file" /><i /><i /><i /><i /><span>{extension}</span></div>
    <div className="document-file-copy"><strong>{filename}</strong><p>ขนาดไฟล์ {size >= 1024 * 1024 ? `${(size / 1024 / 1024).toFixed(1)} MB` : `${Math.ceil(size / 1024)} KB`}</p><span className={`document-status ${status === "Available" ? "is-available" : ""}`}>{status === "Available" && <span aria-hidden="true">✓</span>}{status === "Available" ? "อัปโหลดสำเร็จ" : status}</span></div>
  </div>;
}
