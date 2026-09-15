export function StatusBadge({ status }: { status: string }) {
  const healthy = status === "ok";
  return (
    <span className={`status ${healthy ? "status-ok" : "status-warn"}`}>
      <span aria-hidden="true" className="status-dot" />
      {healthy ? "พร้อมใช้งาน" : "ต้องตรวจสอบ"}
    </span>
  );
}
