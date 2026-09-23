import Link from "next/link";
import Image from "next/image";
import { redirect } from "next/navigation";
import { sessionGET, sessionMultipartPOST } from "@/lib/session-api";

type Profile = { business_type: string; vat_status: string; branch_type: string; name_th: string; name_en: string; tax_id: string; address_1: string; address_2: string; district: string; province: string; postal_code: string; phone: string; has_logo: boolean };
type Membership = { role: "Owner" | "Admin" | "Member"; organization_name: string };
const provinces = "กรุงเทพมหานคร กระบี่ กาญจนบุรี กาฬสินธุ์ กำแพงเพชร ขอนแก่น จันทบุรี ฉะเชิงเทรา ชลบุรี ชัยนาท ชัยภูมิ ชุมพร เชียงราย เชียงใหม่ ตรัง ตราด ตาก นครนายก นครปฐม นครพนม นครราชสีมา นครศรีธรรมราช นครสวรรค์ นนทบุรี นราธิวาส น่าน บึงกาฬ บุรีรัมย์ ปทุมธานี ประจวบคีรีขันธ์ ปราจีนบุรี ปัตตานี พระนครศรีอยุธยา พะเยา พังงา พัทลุง พิจิตร พิษณุโลก เพชรบุรี เพชรบูรณ์ แพร่ ภูเก็ต มหาสารคาม มุกดาหาร แม่ฮ่องสอน ยโสธร ยะลา ร้อยเอ็ด ระนอง ระยอง ราชบุรี ลพบุรี ลำปาง ลำพูน เลย ศรีสะเกษ สกลนคร สงขลา สตูล สมุทรปราการ สมุทรสงคราม สมุทรสาคร สระแก้ว สระบุรี สิงห์บุรี สุโขทัย สุพรรณบุรี สุราษฎร์ธานี สุรินทร์ หนองคาย หนองบัวลำภู อ่างทอง อำนาจเจริญ อุดรธานี อุตรดิตถ์ อุทัยธานี อุบลราชธานี".split(" ");

async function saveBusiness(organization: string, form: FormData) {
  "use server";
  const url = `/o/${encodeURIComponent(organization)}/business`;
  const body = new FormData();
  for (const key of ["business_type", "vat_status", "branch_type", "name_th", "name_en", "tax_id", "address_1", "address_2", "district", "province", "postal_code", "phone"]) body.set(key, String(form.get(key) ?? "").trim());
  const logo = form.get("logo");
  if (logo instanceof File && logo.size > 0) {
    if (logo.size > 512 * 1024) redirect(`${url}?error=logo`);
    body.set("logo", logo);
  }
  const response = await sessionMultipartPOST(`/v1/o/${encodeURIComponent(organization)}/business`, body);
  if (response?.status === 401) redirect("/");
  redirect(`${url}${response?.ok ? "?saved=1" : `?error=${response?.status === 422 ? "invalid" : "save"}`}`);
}

export default async function BusinessPage({ params, searchParams }: { params: Promise<{ organization: string }>; searchParams: Promise<{ saved?: string; error?: string }> }) {
  const { organization } = await params;
  const query = await searchParams;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const [profileResponse, membershipResponse] = await Promise.all([sessionGET(`${base}/business`), sessionGET(base)]);
  if (profileResponse?.status === 401 || membershipResponse?.status === 401) redirect("/");
  if (!profileResponse?.ok || !membershipResponse?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดข้อมูลธุรกิจไม่ได้</h1><Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/business`}>ลองใหม่</Link></section>;
  const profile = await profileResponse.json() as Profile;
  const { membership } = await membershipResponse.json() as { membership: Membership };
  if (!profile.name_th) profile.name_th = membership.organization_name;
  const canEdit = membership.role === "Owner" || membership.role === "Admin";
  const field = (label: string, name: keyof Profile, placeholder = "", help = "", required = false, extra: Record<string, string | number> = {}) => <div className="field"><label htmlFor={name}>{label}{required && " *"}</label><input id={name} name={name} defaultValue={String(profile[name] ?? "")} placeholder={placeholder} required={required} disabled={!canEdit} {...extra} />{help && <p className="field-help">{help}</p>}</div>;
  const select = (label: string, name: keyof Profile, options: [string, string][], required = false) => <div className="field"><label htmlFor={name}>{label}{required && " *"}</label><select id={name} name={name} defaultValue={String(profile[name] ?? "")} required={required} disabled={!canEdit}><option value="">เลือก{label}</option>{options.map(([value, text]) => <option value={value} key={value}>{text}</option>)}</select></div>;
  return <section className="workspace-page business-page">
    <div className="work-header"><div><p className="eyebrow">ข้อมูล Organization</p><h1>การจัดการธุรกิจ</h1><p className="intro">ข้อมูลที่ใช้แสดงบนเอกสารและช่วยตรวจข้อมูลใบเสร็จ</p></div></div>
    {query.saved && <p className="success-message" role="status">บันทึกการเปลี่ยนแปลงแล้ว</p>}
    {query.error && <p className="form-error" role="alert">{query.error === "logo" ? "โลโก้ต้องมีขนาดไม่เกิน 512 KB" : query.error === "invalid" ? "ข้อมูลไม่ถูกต้อง กรุณาตรวจสอบช่องที่กรอกและชนิดไฟล์โลโก้" : "ยังบันทึกข้อมูลไม่ได้ กรุณาลองอีกครั้ง"}</p>}
    <form action={saveBusiness.bind(null, organization)} className="card work-form business-form" encType="multipart/form-data">
      <div className="business-heading"><div><h2>ข้อมูลธุรกิจ</h2><p className="field-help">{membership.organization_name}</p></div><span className="status-pill">{membership.role}</span></div>
      <div className="business-grid">
        {select("ประเภทธุรกิจ", "business_type", ["บริษัทจำกัด", "ห้างหุ้นส่วนจำกัด", "ห้างหุ้นส่วนสามัญ", "ห้างหุ้นส่วนสามัญนิติบุคคล", "บริษัทมหาชนจำกัด", "ร้านค้า/กิจการเจ้าของคนเดียว", "คณะบุคคล", "มูลนิธิ", "สมาคม", "สหกรณ์", "บุคคลธรรมดา/ฟรีแลนซ์"].map((x) => [x, x]), true)}
        {select("สถานะการจด VAT", "vat_status", [["registered", "จด VAT แล้ว"], ["unregistered", "ยังไม่จด VAT"]], true)}
        {select("ประเภทสาขา", "branch_type", [["head", "สำนักงานใหญ่"], ["branch", "สาขา"], ["none", "ไม่มี"]], true)}
        {field("ชื่อธุรกิจภาษาไทย", "name_th", "เช่น บริษัท เพย์เปอร์ จำกัด", "ใส่ชื่อเต็มโดยไม่ต้องใส่ข้อมูลสาขา เลือกที่ช่องประเภทสาขาเท่านั้น", true, { maxLength: 160 })}
        {field("ชื่อธุรกิจภาษาอังกฤษ", "name_en", "เช่น Paypers Co., Ltd.", "ใส่ชื่อเต็มภาษาอังกฤษ", false, { maxLength: 160 })}
        {field("เลขที่ผู้เสียภาษี", "tax_id", "เช่น 0123456789012", "แนะนำให้ใส่เลขที่ผู้เสียภาษีเพื่อช่วยให้การอ่านใบเสร็จแม่นขึ้น", false, { inputMode: "numeric", pattern: "[0-9]{13}", maxLength: 13 })}
        {field("ที่อยู่บรรทัดที่ 1", "address_1", "เช่น 123 ถ.สุขุมวิท", "", false, { maxLength: 240 })}
        {field("ที่อยู่บรรทัดที่ 2", "address_2", "เช่น แขวงคลองเตย (ทางเลือก)", "", false, { maxLength: 240 })}
        {field("อำเภอ/เขต", "district", "เช่น เขตคลองเตย", "", false, { maxLength: 120 })}
        {select("จังหวัด", "province", provinces.map((x) => [x, x]))}
        {field("รหัสไปรษณีย์", "postal_code", "เช่น 10110", "", false, { inputMode: "numeric", pattern: "[0-9]{5}", maxLength: 5 })}
        {field("เบอร์โทรติดต่อ", "phone", "เช่น 0635167015", "", true, { inputMode: "tel", pattern: "[0-9]{9,10}", maxLength: 10 })}
      </div>
      <div className="business-logo"><div><label htmlFor="logo">โลโก้บริษัท</label><p className="field-help">PNG, JPEG หรือ WebP ขนาดไม่เกิน 512 KB</p>{profile.has_logo && <Image unoptimized src={`/api/o/${encodeURIComponent(organization)}/business/logo`} alt="โลโก้บริษัทปัจจุบัน" width={96} height={96} />}</div>{canEdit && <input id="logo" name="logo" type="file" accept="image/png,image/jpeg,image/webp" />}</div>
      {canEdit && <div className="business-actions"><Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/tasks`}>ยกเลิก</Link><button className="button" type="submit">บันทึกการเปลี่ยนแปลง</button></div>}
    </form>
  </section>;
}
