"use client";

export default function ErrorPage({ reset }: { reset: () => void }) {
  return <div role="alert"><p className="eyebrow">เชื่อมต่อไม่สำเร็จ</p><h1>ยังโหลดภาพรวมไม่ได้</h1><p className="intro">ตรวจสอบ API หรือฐานข้อมูล แล้วลองใหม่อีกครั้ง</p><button className="button" onClick={reset}>ลองใหม่</button></div>;
}
