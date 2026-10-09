"use client";

import { useState, type ReactNode } from "react";
import Link from "next/link";
import { DocumentIcon, DocumentStatus } from "./document-visuals";
import { workflowStages, type WorkflowStage } from "@/lib/document-workflow";
import type { InboxDocument } from "./document-table";
import "./workflow.css";

type WorkflowDocument = InboxDocument & { stage: WorkflowStage; typeLabel: string; ocrDisabled: boolean };
export default function WorkflowInbox({ documents, organization, upload, tools, search, pagination, notices, canManage, changeStatus }: {
  documents: WorkflowDocument[]; organization: string; upload: ReactNode; tools: ReactNode; search: ReactNode; pagination: ReactNode; notices: ReactNode; canManage: boolean;
  changeStatus: (organization: string, document: string, action: "trash" | "restore") => Promise<void>;
}) {
  const [filter, setFilter] = useState<WorkflowStage | "all">("all");
  const [uploadOpen, setUploadOpen] = useState(false);
  const root = `/o/${encodeURIComponent(organization)}/documents`;
  const href = (id: string) => `${root}/${encodeURIComponent(id)}`;
  const shown = documents.filter(doc => filter === "all" || doc.stage === filter);
  const pending = documents.filter(doc => doc.stage === "review");
  const count = (stage: WorkflowStage) => documents.filter(doc => doc.stage === stage).length;
  return <section className="workflow-inbox">
    <p className="workflow-context">พื้นที่ทำงาน / เอกสาร</p>
    <header className="workflow-header"><div><p className="eyebrow">DOCUMENT WORKSPACE</p><h1>เอกสาร <span>{documents.length}</span></h1><p>จากไฟล์ต้นฉบับ สู่ข้อมูลที่คุณตรวจแล้ว — ทุกขั้นตอนในที่เดียว</p></div><button className="button" aria-expanded={uploadOpen} aria-controls="workflow-upload" onClick={() => setUploadOpen(!uploadOpen)}><DocumentIcon kind="upload" />เพิ่มเอกสาร</button></header>
    <div id="workflow-upload" hidden={!uploadOpen}>{upload}</div>
    <nav className="workflow-pipeline" aria-label="ขั้นตอนเอกสาร">{(["waiting", "processing", "review", "confirmed"] as const).map((key, i) => <button key={key} aria-pressed={filter === key} onClick={() => setFilter(filter === key ? "all" : key)}><span className={`workflow-step ${key}`}>{String(i + 1).padStart(2, "0")}</span><span><strong>{workflowStages[key].label}</strong><small>{workflowStages[key].hint}</small></span><b>{count(key)}</b></button>)}</nav>
    <p className="workflow-count-note">จำนวนและขั้นตอนแสดงเฉพาะเอกสารในหน้านี้ · ใช้ค้นหาเพื่อดูเอกสารอื่น</p>
    {pending.length > 0 && <div className="workflow-priority"><div><strong>มี {pending.length} เอกสารรอคุณตรวจข้อมูล</strong><p>ตรวจประเภทเอกสารและค่าที่อ่านได้เทียบกับต้นฉบับก่อนยืนยัน</p></div><Link className="text-button" href={href(pending[0].id)}>เริ่มตรวจ →</Link></div>}
    {notices}
    <div className="workflow-tools">{search}{tools}</div>
    <nav className="workflow-filters" aria-label="กรองขั้นตอนในหน้านี้">{(["all", "review", "error", "unavailable", "inactive"] as const).filter(key => key === "all" || key === "review" || count(key) > 0).map(key => <button key={key} aria-pressed={filter === key} onClick={() => setFilter(key)}>{key === "all" ? "ทั้งหมด" : workflowStages[key].label}<span>{key === "all" ? documents.length : count(key)}</span></button>)}</nav>
    <div className="workflow-table-wrap"><table className="workflow-table"><caption className="sr-only">เอกสารในหน้านี้และขั้นตอนการตรวจข้อมูล</caption><thead><tr><th scope="col">เอกสาร</th><th scope="col">ช่องทาง</th><th scope="col">รับเมื่อ</th><th scope="col">ขั้นตอน</th><th scope="col">การดำเนินการ</th></tr></thead><tbody>{shown.map(doc => <tr key={doc.id}><th scope="row"><Link className="workflow-file" href={href(doc.id)}><span><DocumentIcon kind="file" /></span><div><strong>{doc.filename}</strong><small>{doc.typeLabel} · {(doc.size / 1024 / 1024).toFixed(2)} MiB</small></div></Link></th><td data-label="ช่องทาง">{({ Web: "อัปโหลด", LINE: "LINE", Drive: "Google Drive" } as Record<string, string>)[doc.source_channel ?? ""] ?? doc.source_channel ?? "—"}</td><td data-label="รับเมื่อ">{new Intl.DateTimeFormat("th-TH", { dateStyle: "medium", timeZone: "Asia/Bangkok" }).format(new Date(doc.accepted_at))}</td><td data-label="ขั้นตอน"><div><span className={`workflow-badge ${doc.stage}`}>{workflowStages[doc.stage].label}</span>{doc.ocrDisabled && <small>ระบบ OCR ยังไม่เปิดใช้งาน</small>}{!["Available", "Trash"].includes(doc.status) && <small><DocumentStatus status={doc.status} /></small>}</div></td><td><div className="workflow-row-actions"><Link className="secondary-button" href={href(doc.id)}>{workflowStages[doc.stage].action} ↗</Link>{doc.status !== "Trash" && <a className="text-button" aria-label={`ดาวน์โหลด ${doc.filename}`} href={`/api${root}/${encodeURIComponent(doc.id)}/original`}><DocumentIcon kind="download" /></a>}{canManage && <form action={changeStatus.bind(null, organization, doc.id, doc.status === "Trash" ? "restore" : "trash")}><button className="text-button" aria-label={`${doc.status === "Trash" ? "กู้คืน" : "ย้ายไปถังขยะ"} ${doc.filename}`} type="submit">{doc.status === "Trash" ? "กู้คืน" : <DocumentIcon kind="trash" />}</button></form>}</div></td></tr>)}</tbody></table>{shown.length === 0 && <div className="empty-state"><h2>ไม่พบเอกสารในขั้นตอนนี้</h2><p>เลือกทั้งหมด เปลี่ยนคำค้น หรือเพิ่มเอกสารเพื่อเริ่มต้น</p><button className="secondary-button" onClick={() => setFilter("all")}>ดูทั้งหมดในหน้านี้</button></div>}<footer className="workflow-table-footer"><span>แสดง {shown.length} จาก {documents.length} เอกสารในหน้านี้</span>{pagination}</footer></div>
    <p className="workflow-count-note">ข้อมูลที่ AI อ่านได้เป็นฉบับร่าง จนกว่าคุณจะตรวจและยืนยัน · ต้นฉบับเก็บไว้เสมอ</p>
  </section>;
}
