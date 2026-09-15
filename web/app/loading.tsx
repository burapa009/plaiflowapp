export default function Loading() {
  return <><p className="eyebrow">กำลังอัปเดต</p><h1>ภาพรวม Inbound Event</h1><div className="grid metrics" aria-label="กำลังโหลดข้อมูล">{Array.from({ length: 5 }, (_, i) => <div className="card skeleton" key={i} />)}</div></>;
}
