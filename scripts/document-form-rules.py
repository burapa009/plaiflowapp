"""Build the shared, versioned field catalogue and its reviewable rules matrix.

Run from this checkout. Legal dates left null are deliberately unverified;
effectiveFrom is the PlaiFlow rule activation date, not a claimed statute date.
"""
import json
from pathlib import Path

root = Path(__file__).resolve().parents[1]
path = root / 'api/internal/extraction/document-forms.json'
c = json.loads(path.read_text(encoding='utf-8'))
version = '2026-10-05.1'
workflow_required = {'receipt': [], 'tax_invoice': [], 'tax_invoice_receipt': [], 'invoice': ['document_number', 'issue_date', 'seller_name', 'buyer_name', 'total_amount', 'items.description'], 'billing_note': ['document_number', 'issue_date', 'seller_name', 'buyer_name', 'total_amount', 'references.document_number', 'references.allocated'], 'delivery_note': [], 'receipt_voucher': ['issue_date', 'seller_name', 'buyer_name', 'total_amount', 'purpose'], 'receipt_substitute': ['total_amount', 'prepared_by', 'expense_date', 'purpose', 'reason_no_receipt', 'evidence_note'], 'payment_voucher': ['issue_date', 'buyer_name', 'total_amount', 'purpose'], 'abbreviated_tax_invoice': [], 'credit_note': [], 'debit_note': [], 'withholding_tax_certificate': [], 'bank_slip': ['total_amount', 'payment_date', 'payment_reference'], 'bank_document': ['issue_date', 'total_amount', 'bank_name', 'bank_account', 'bank_document_type', 'transaction_description'], 'bank_statement': ['currency', 'bank_account', 'period_from', 'period_to', 'transactions.date', 'transactions.description'], 'unknown': ['description', 'evidence_note', 'other_subtype'], 'pre_receipt': ['issue_date'], 'quotation': ['issue_date'], 'purchase_order': ['issue_date'], 'expense_claim': ['issue_date'], 'petty_cash': ['issue_date']}
def field(key, label, kind='text', options=None):
    c['fields'][key] = dict(label=label, kind=kind)
    if options: c['fields'][key]['options'] = options
tri = [['unknown','ยังไม่ทราบ / ต้องตรวจสอบ'],['yes','ใช่'],['no','ไม่ใช่']]
for key,label in [('buyer_vat_registered','ผู้ซื้อจด VAT'),('buyer_tax_required','ผู้ซื้อต้องมีเลขภาษี'),('seller_vat_registered','ผู้ออกจด VAT'),('receipt_105','ใบรับอยู่ในบังคับมาตรา 105'),('has_book','จัดทำเป็นเล่ม'),('agent_issued','ออกผ่านตัวแทน'),('special_issuance','ออกแทน / ยกเลิก / รูปแบบเฉพาะ'),('track_debt','ใช้ติดตามหนี้'),('record_payment','บันทึกรับหรือจ่ายเงินจริง'),('reconcile_period','ตรวจยอดครบช่วง'),('receipt_goods_detail','ใบรับต้องระบุสินค้าและผู้ซื้อเฉพาะกรณี')]: field(key,label,'choice',tri)
field('delivery_scope','หน้าที่ใบส่งของ','choice',[['unknown','ยังไม่ทราบ / ต้องตรวจสอบ'],['section_105_quater','ขายสินค้าตามมาตรา 105 จัตวา'],['internal','ส่งมอบ / เคลื่อนย้ายภายใน']])
field('adjustment_scope','หน้าที่ใบปรับหนี้','choice',[['unknown','ยังไม่ทราบ / ต้องตรวจสอบ'],['vat','ปรับหนี้ทาง VAT'],['general','ปรับหนี้ทั่วไป']])
field('payment_method','วิธีชำระ','choice',[['cash','เงินสด'],['transfer','โอนเงิน'],['cheque','เช็ค'],['card','บัตร'],['other','อื่น ๆ']])
field('bank_document_type','ประเภทเอกสารธนาคาร','choice',[['deposit','ใบฝาก'],['withdrawal','ใบถอน'],['fee','ค่าธรรมเนียม'],['other','อื่น ๆ']])
field('document_title','ชื่อเอกสารตามต้นฉบับ')
field('book_number','เลขเล่ม')
field('vat_included_text','ข้อความแสดงว่าราคารวม VAT ตามต้นฉบับ')
field('agent_details','ข้อมูลตัวแทน','textarea')
field('special_issuance_text','ข้อความออกแทน / ยกเลิก / รูปแบบเฉพาะ','textarea')
field('signature_evidence','ข้อมูลการลงนามตามต้นฉบับ (ไม่สร้างลายเซ็น)','textarea')
field('evidence_note','รายละเอียดหลักฐานประกอบ / ที่อยู่หลักฐาน','textarea')
field('other_subtype','ประเภทย่อยเอกสาร')
field('posting_date','วันที่ลงบัญชี','date')
field('debit_account','บัญชีเดบิต')
field('credit_account','บัญชีเครดิต')
field('posting_amount','จำนวนเงินลงบัญชี','money')
def section(key,label,fields): c['sections'][key]=dict(label=label,fields=fields.split())
section('document_identity','ข้อความบนเอกสาร','document_title has_book book_number evidence_note')
section('vat_context','เงื่อนไขภาษีที่ต้องตรวจสอบ','seller_vat_registered buyer_vat_registered buyer_tax_required agent_issued agent_details special_issuance special_issuance_text')
section('receipt_context','เงื่อนไขใบรับ','receipt_105 receipt_goods_detail record_payment')
section('delivery_context','หน้าที่ใบส่งของ','delivery_scope')
section('adjustment_context','หน้าที่ใบปรับหนี้','adjustment_scope')
section('wht_context','ข้อมูลตามแบบต้นฉบับ','signature_evidence special_issuance special_issuance_text')
section('abbreviated_context','ข้อความราคารวม VAT','vat_included_text')
section('debt_context','การติดตามหนี้','track_debt')
section('reconcile_context','ขอบเขตกระทบยอด','reconcile_period')
section('payment_context','การบันทึกเงิน','record_payment')
section('other_context','ลักษณะเอกสาร','other_subtype')
# Included as additional accounting data, never presented as OCR evidence.
section('posting','ข้อมูลเพิ่มเติมเพื่อบัญชี (ไม่ใช่ข้อความบนต้นฉบับ)','posting_date debit_account credit_account posting_amount')
def append_column(table,column):
    if column[0] not in [x[0] for x in c['tables'][table]['columns']]: c['tables'][table]['columns'].append(column)
append_column('deliveries',['unit_price','ราคาสินค้า','money'])
append_column('deliveries',['amount','มูลค่าสินค้า','money'])
append_column('references',['evidence','หลักฐานต้นทางภายนอก','text'])
for name,t in c['types'].items():
    additions=['document_identity','posting']
    if name in ['tax_invoice','tax_invoice_receipt']: additions+=['vat_context']
    if name in ['receipt','tax_invoice_receipt']: additions+=['receipt_context']
    if name=='delivery_note': additions+=['delivery_context']
    if name in ['credit_note','debit_note']: additions+=['adjustment_context','vat_context']
    if name=='withholding_tax_certificate': additions+=['wht_context']
    if name=='abbreviated_tax_invoice': additions+=['abbreviated_context']
    if name in ['invoice','billing_note','pre_receipt']: additions+=['debt_context']
    if name=='bank_statement': additions+=['reconcile_context']
    if name in ['receipt_voucher','payment_voucher']: additions+=['payment_context']
    if name=='unknown': additions+=['other_context']
    for s in additions:
        if s not in t['sections']: t['sections'].append(s)

allkeys=list(c['fields'])+[tab+'.'+x[0] for tab,v in c['tables'].items() for x in v['columns']]
rules={}
for name,t in c['types'].items():
    active={k for s in t['sections'] for k in c['sections'][s]['fields']}
    active.update(tab+'.'+x[0] for tab in t['tables'] for x in c['tables'][tab]['columns'])
    rules[name]={k:dict(field=k,requirement='OPTIONAL' if k in active else 'NOT_APPLICABLE',condition={},conditionLabel='',action=[],reason='ข้อมูลเพิ่มเติมตามหลักฐาน' if k in active else 'ไม่เกี่ยวข้องกับหน้าที่เอกสารนี้',sourceUrl='system_policy',legalReference='',effectiveFrom='2026-10-04',legalEffectiveFrom=None,ruleVersion=version) for k in allkeys}
def setrule(types,keys,requirement='ACCOUNTING_REQUIRED',condition=None,label='',source='system_policy',legal='',actions=None):
    for name in types.split():
        for key in keys.split():
            r=rules[name][key]
            if r['requirement']=='NOT_APPLICABLE': continue
            next_rule=dict(r,requirement=requirement,condition=condition or {},conditionLabel=label,action=actions if actions is not None else (['issue','tax'] if source!='system_policy' else ['account']),reason=('ตรวจรายการตาม '+legal+'; ไม่ใช่ผลรับรองสิทธิภาษี') if legal else 'ข้อมูลขั้นต่ำสำหรับ workflow บัญชีของ PlaiFlow (นโยบายระบบ)',sourceUrl=source,legalReference=legal)
            next_rule.pop('additionalRules',None)
            if r['requirement'] not in ['OPTIONAL','NOT_APPLICABLE']:
                primary = {k:v for k,v in r.items() if k != 'additionalRules'}
                if next_rule not in [primary, *r.get('additionalRules',[])]:
                    r.setdefault('additionalRules',[]).append(next_rule)
            else: rules[name][key]=next_rule
full='tax_invoice tax_invoice_receipt'
vat='https://www.rd.go.th/5208.html'; receipt='https://www.rd.go.th/5203.html'; branch='https://www.rd.go.th/3400.html'; wht='https://www.rd.go.th/3171.html'
setrule(full,'document_title document_number issue_date seller_name seller_tax_id seller_address buyer_name buyer_address items.description items.quantity items.amount vat_amount','LEGAL_REQUIRED',source=vat,legal='มาตรา 86/4')
setrule(full,'buyer_tax_id','CONDITIONAL_REQUIRED',{'buyer_vat_registered':'yes'},'ผู้ซื้อจด VAT',branch,'ประกาศ VAT 39 ข้อ 7 แก้ไขโดยฉบับ 199')
setrule(full,'buyer_tax_id','CONDITIONAL_REQUIRED',{'buyer_tax_required':'yes'},'มีหน้าที่ต้องใช้เลขภาษีตามกรณีที่ผู้ตรวจระบุ')
setrule(full,'seller_branch','CONDITIONAL_REQUIRED',{'seller_vat_registered':'yes'},'ผู้ออกจด VAT',branch,'ประกาศ VAT 39 ข้อ 8 และฉบับแก้ไข')
setrule(full,'buyer_branch','CONDITIONAL_REQUIRED',{'buyer_vat_registered':'yes'},'ผู้ซื้อจด VAT',branch,'ประกาศ VAT 39 ข้อ 9 และฉบับแก้ไข')
for name in full.split():
    for key in ['buyer_tax_id','seller_branch','buyer_branch']:
        rules[name][key]['legalEffectiveFrom']='2015-01-01'
setrule(full,'agent_details','CONDITIONAL_REQUIRED',{'agent_issued':'yes'},'ออกผ่านตัวแทน',vat,'มาตรา 86/4 (2)')
setrule(full,'special_issuance_text','CONDITIONAL_REQUIRED',{'special_issuance':'yes'},'ออกแทน / ยกเลิก / รูปแบบเฉพาะ',branch,'ประกาศ VAT 39 และฉบับแก้ไข; ต้องตรวจกรณีเฉพาะ')
setrule('abbreviated_tax_invoice','document_title seller_name seller_tax_id document_number issue_date items.description items.quantity items.amount vat_included_text','LEGAL_REQUIRED',source=vat,legal='มาตรา 86/6')
setrule('receipt tax_invoice_receipt','seller_name seller_tax_id document_number issue_date paid_amount','CONDITIONAL_REQUIRED',{'receipt_105':'yes'},'อยู่ในบังคับมาตรา 105 และ 105 ทวิ',receipt,'มาตรา 105 และ 105 ทวิ')
setrule('receipt tax_invoice_receipt','buyer_name buyer_address items.description items.quantity','CONDITIONAL_REQUIRED',{'receipt_goods_detail':'yes'},'ใบรับเข้ากรณีต้องระบุสินค้าและผู้ซื้อ',receipt,'มาตรา 105 ทวิ (6)')
setrule('delivery_note','seller_name seller_tax_id buyer_name document_number issue_date deliveries.description deliveries.quantity deliveries.unit_price','CONDITIONAL_REQUIRED',{'delivery_scope':'section_105_quater'},'ขายสินค้าตามมาตรา 105 จัตวา',receipt,'มาตรา 105 จัตวา')
setrule('delivery_note','seller_name buyer_name issue_date deliveries.description deliveries.delivered','CONDITIONAL_REQUIRED',{'delivery_scope':'internal'},'ส่งมอบ / เคลื่อนย้ายภายใน')
for name,article in [('credit_note','86/10'),('debit_note','86/9')]:
    setrule(name,'document_title document_number issue_date seller_name seller_tax_id seller_address buyer_name buyer_address references.document_number before_amount after_amount total_amount vat_amount adjustment_reason','CONDITIONAL_REQUIRED',{'adjustment_scope':'vat'},'ปรับหนี้ทาง VAT',vat,'มาตรา '+article)
setrule('withholding_tax_certificate','document_number issue_date seller_name seller_address seller_tax_id buyer_name buyer_address buyer_tax_id income.description income.payment_date income.amount income.tax withholding_condition signature_evidence','LEGAL_REQUIRED',source=wht,legal='มาตรา 50 ทวิ; ประกาศ 62 และฉบับแก้ไข')
for name,t in c['types'].items():
    for k in workflow_required[name]:
        if rules[name][k]['requirement']=='OPTIONAL': setrule(name,k)
    setrule(name,'book_number','CONDITIONAL_REQUIRED',{'has_book':'yes'},'จัดทำเป็นเล่ม',vat if name in full.split()+['abbreviated_tax_invoice'] else (wht if name=='withholding_tax_certificate' else receipt if name in ['receipt','delivery_note'] else 'system_policy'),'เลขเล่มเมื่อมี' if name in full.split()+['abbreviated_tax_invoice','receipt','delivery_note','withholding_tax_certificate'] else '')
    setrule(name,'posting_date debit_account credit_account posting_amount')
    if 'references' in t['tables']:
        setrule(name,'references.allocated','CONDITIONAL_REQUIRED',{'$row.document_id':'@present'},'จับคู่กับเอกสารในทีม')
    for k in ['payment_reference','bank_name','bank_account']:
        if k in rules[name] and rules[name][k]['requirement']!='NOT_APPLICABLE':
            rules[name][k]['hiddenWhen']={'payment_method':'cash'}
    # No input is legally required just because OCR is being acknowledged.
    t['required']=[]
setrule('receipt tax_invoice_receipt','paid_amount')
setrule('invoice','seller_name buyer_name document_number issue_date items.description total_amount')
setrule('billing_note','seller_name buyer_name issue_date document_number references.document_number references.allocated total_amount')
setrule('receipt_voucher','seller_name buyer_name issue_date purpose total_amount')
setrule('payment_voucher','buyer_name issue_date purpose total_amount')
setrule('receipt_substitute','prepared_by expense_date purpose total_amount reason_no_receipt evidence_note')
setrule('bank_slip','payment_date total_amount payment_reference')
setrule('bank_document','bank_document_type bank_name bank_account issue_date transaction_description total_amount')
setrule('bank_statement','bank_account period_from period_to currency transactions.date transactions.description')
setrule('unknown','other_subtype description evidence_note')
setrule('quotation purchase_order expense_claim petty_cash pre_receipt','issue_date')
setrule('invoice billing_note pre_receipt','due_date','CONDITIONAL_REQUIRED',{'track_debt':'yes'},'ใช้ติดตามหนี้ตามนโยบายระบบ')
setrule('receipt tax_invoice_receipt receipt_voucher payment_voucher','payment_method','CONDITIONAL_REQUIRED',{'record_payment':'yes'},'บันทึกรับหรือจ่ายเงินจริง')
setrule('bank_statement','opening_balance closing_balance','CONDITIONAL_REQUIRED',{'reconcile_period':'yes'},'ตรวจยอดครบช่วง')
for name in rules:
    for k in ['posting_date','debit_account','credit_account','posting_amount']: rules[name][k]['evidenceCheck']=False
    for k in ['deliveries.unit_price','deliveries.amount']:
        rules[name][k]['hiddenWhen']={'delivery_scope':'internal'}
c['ruleVersion']=version
c['rules']=rules
for out in [path,root/'web/lib/document-forms.json']:
    out.write_text(json.dumps(c,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
lines=['# กฎช่องเอกสารบัญชี '+version,'','Generated by scripts/document-form-rules.py. effectiveFrom = วันเริ่มใช้กฎระบบ ไม่ใช่วันมีผลกฎหมาย; legalEffectiveFrom=null คือยังไม่ได้ยืนยันวันที่ย้อนหลัง. ไม่รองรับการตัดสินสิทธิภาษี/ออกเอกสารทางการ.','','| ประเภท | ช่อง | สถานะ | เงื่อนไข | action | เหตุผล | แหล่งอ้างอิง |','|---|---|---|---|---|---|---|']
for name,rr in rules.items():
    for k,primary in rr.items():
        for r in [primary, *primary.get('additionalRules',[])]:
            lines.append('| '+' | '.join([c['types'][name]['label'],k,r['requirement'],r['conditionLabel'] or '—',', '.join(r['action']) or '—',r['reason'],r['sourceUrl']])+' |')
(root/'docs/runbooks/accounting-document-field-rules.md').write_text('\n'.join(lines)+'\n',encoding='utf-8')
