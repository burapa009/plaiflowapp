import "server-only";
import { sessionGET } from "@/lib/session-api";
import { documentWorkflow } from "@/lib/document-workflow";
import { formConfig, type DocumentFormResponse } from "@/lib/document-form";
import type { InboxDocument } from "./document-table";

export async function workflowDocument(organization: string, doc: InboxDocument) {
  const path = `/v1/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(doc.id)}`;
  const response = ["Available", "Archived"].includes(doc.status) ? await sessionGET(`${path}/ocr`) : null;
  const ocr = response?.ok ? (await response.json() as { ocr: { status: string; enabled: boolean } }).ocr : null;
  const formResponse = ocr?.status === "Completed" ? await sessionGET(`${path}/form`) : null;
  const form = formResponse?.ok ? await formResponse.json() as DocumentFormResponse : null;
  return { ...doc, stage: documentWorkflow(doc.status, ocr?.status, form ? { status: form.form.status, source_superseded: form.source_superseded } : undefined),
    typeLabel: form ? formConfig.types[form.form.data.type]?.label ?? "ยังไม่ระบุประเภท" : "ยังไม่มีข้อมูล",
    ocrDisabled: ocr?.enabled === false && ocr.status === "NotScheduled",
  };
}
