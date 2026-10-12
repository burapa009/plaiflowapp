import { fieldVisible, formConfig, type CanonicalDocument } from "./document-form.ts";

export type OCRFieldEvidence = {
  presence: string; raw: string; normalized: string; confidence: string;
  evidence: { page: number; line: number; ocr_confidence?: number }[];
};
export type FormEvidence = { jobId: string; fields: Record<string, OCRFieldEvidence> };

// Coordinates belong to one OCR run. Never attach newer evidence to an older draft.
export function fieldEvidence(data: CanonicalDocument, key: string, evidence?: FormEvidence) {
  const source = data.source?.fields[key];
  const stale = !!data.source?.ocr_job_id && !!evidence?.jobId && data.source.ocr_job_id !== evidence.jobId;
  const field = data.source?.ocr_job_id && data.source.ocr_job_id === evidence?.jobId ? evidence.fields[key] : undefined;
  const value = data.fields[key] ?? "";
  const changed = source !== undefined && value !== source;
  const uncertain = field?.presence === "ambiguous" || !!field?.evidence?.some(ref => ref.ocr_confidence !== undefined && ref.ocr_confidence < .8);
  return {
    source, raw: field?.raw, locations: (field?.evidence ?? []).filter(ref => Number.isInteger(ref.page) && ref.page > 0 && Number.isInteger(ref.line) && ref.line > 0),
    changed, uncertain, stale,
    status: changed ? "แก้จากค่า OCR" : uncertain ? "ควรตรวจต้นฉบับ" : source?.trim() ? stale ? "OCR รอบก่อน" : "อ่านจากเอกสาร" : value.trim() ? "ข้อมูลในแบบฟอร์ม" : "ยังไม่มีข้อมูล",
  };
}

export function visibleFormFields(data: CanonicalDocument): string[] {
  const type = formConfig.types[data.type] ?? formConfig.types.unknown;
  return [...new Set(type.sections.filter(section => section !== "posting").flatMap(section => formConfig.sections[section].fields))].filter(key => fieldVisible(data, key));
}

export function formFieldLabel(data: CanonicalDocument, path: string): string {
  const type = formConfig.types[data.type] ?? formConfig.types.unknown;
  const [table, index, column] = path.split(".");
  const tableConfig = formConfig.tables[table];
  if (tableConfig) {
    const key = column ?? index;
    const label = tableConfig.columns.find(([name]) => name === key)?.[1];
    return `${tableConfig.label}${column ? ` แถว ${Number(index) + 1}` : ""}${label ? ` · ${label}` : ""}`;
  }
  return (formConfig.fields[path]?.label ?? path).replace("ฝ่ายที่หนึ่ง", type.roles[0]).replace("ฝ่ายที่สอง", type.roles[1]);
}
