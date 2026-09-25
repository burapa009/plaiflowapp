# Phase 11 billing UI draft

Status: Paused, 2026-09-25. Stripe was canceled and no provider is selected. This is a historical design draft; checkout details require revision before implementation resumes.

## Public pricing

- A single three-option selector changes every card together: รายเดือน / ทุก 6 เดือน / รายปี. Default to monthly. Preserve the selected interval in the URL and when moving to the authenticated checkout review.
- Stack Free, Starter, and Business cards vertically at 375 px; use a three-column comparison only when space allows. Put Accounting Firm in a separate clearly named section. Show its own 1,000-Document monthly grant, 5 LINE groups, 5 staff seats, and 20 client relationships; explain that each client has its own allowance, LINE limit, and Scan Credits. Until Phase 10 activation gates pass, show its approved price with a plainly unavailable purchase action. No horizontal card carousel.
- Each paid card leads with **ยอดชำระ ฿… / รอบ …** and a plain renewal sentence, then **เฉลี่ย ฿…/เดือน** as secondary comparison. Use the approved Plan/interval totals in `phase-11-decisions.md`: 5% off six monthly payments and 15% off twelve monthly payments. Show the exact next renewal interval and actual savings; do not make the effective monthly amount look like the charge.

| Plan | รายเดือน: จ่ายจริง / เฉลี่ยต่อเดือน | ทุก 6 เดือน: จ่ายจริง / เฉลี่ยต่อเดือน | รายปี: จ่ายจริง / เฉลี่ยต่อเดือน |
|---|---:|---:|---:|
| Starter | ฿150 / ฿150 | ฿855 / ฿142.50 | ฿1,530 / ฿127.50 |
| Business | ฿250 / ฿250 | ฿1,425 / ฿237.50 | ฿2,550 / ฿212.50 |
| Accounting Firm | ฿1,000 / ฿1,000 | ฿5,700 / ฿950 | ฿10,200 / ฿850 |
- Keep Business's “แนะนำ” label as a modest border and text badge. Do not preselect a paid plan, obscure Free, use countdown pressure, or make the recommendation a different sized purchase button.
- Show decisive limits in the same order on every card: users, LINE groups, new Document allowance/month, zero included scan pages, and included capabilities. On paid cards, explain that unused monthly allowance carries forward for up to two years; on Free, say unused allowance expires each month. Explain that no automatic overage is charged and intake stops when available Document allowance is exhausted. Do not imply purchased scan credits increase Document allowance.
- One primary action per card. Owner enters an authenticated review step; Admin sees the Plan and renewal facts but no checkout action. While the seller is not VAT registered, show the approved price as the full charge without a VAT line or Thai tax-invoice promise.

## Checkout review and handoff

- Before redirecting to Stripe, show Organization name, Plan, interval, full charge, renewal terms, and what changes now versus at renewal. An interval change is editable here without losing the Plan selection. Require a verified billing-contact email and an Owner-reviewed buyer name, address, and tax ID for the seller receipt.
- The action says “ไปชำระเงินกับ Stripe” and identifies Stripe as the payment page. Show a return path to the review page. Disable repeat submission and show progress while creating the Checkout Session.
- Returning from Stripe shows “กำลังตรวจสอบการชำระเงิน” until the local verified subscription state changes. A success redirect alone never displays a paid entitlement or a final success claim. An incomplete, expired, or canceled session offers a safe retry without creating duplicate subscriptions.

## Organization billing page

- Lead with one status panel: current Plan, billing interval, full recurring charge, next charge or access-end date, and status text. Place “เปลี่ยนบัตร” and Stripe invoice/payment history behind Owner access to a restricted Stripe Portal session. Show the seller's own ordinary receipts as a distinct, tenant-authorized downloadable history; do not label either document a Thai VAT tax invoice. Billing Profile edits affect future receipts only; correction of an issued receipt goes through support.
- Show new monthly Document allowance, carried balance (paid Plans only), accepted Documents, and available allowance as distinct figures. Label the next monthly allowance grant date and earliest carried-grant expiry, separately from the subscription renewal date. Show an in-app reminder when a positive amount expires within 30 days. On Free, show the monthly reset date instead of a carried balance; if payment recovery is available, show suspended carry and its recovery deadline as unavailable. On a paid downgrade, preview the amount of carried allowance removed by the destination Plan's 24-month ceiling. Show users and LINE groups against their limits.
- Owner actions are “เปลี่ยนแพ็กเกจหรือรอบ”, “ยกเลิกการต่ออายุ”, and “กลับมาต่ออายุ”; separate cancellation visually from the primary action. Admin sees status and renewal date only.
- For an upgrade, preview unused-time credit, amount due now, resulting Plan, added Document allowance for the current month, and new renewal date before confirmation. For a downgrade or shorter interval, show the scheduled effective date and retained current rights until then.
- For scheduled cancellation, show “ยังใช้ได้ถึง …” and one clear undo action. Confirmation repeats the Organization, paid-through date, and loss of paid capabilities after that date. Do not require a cancellation reason or present retention offers that block the action.

## Scan-credit packs

- Show the four revised packs separately from recurring Plans: 500 pages ฿219, 800 pages ฿339, 1,800 pages ฿699, and 5,000 pages ฿1,899. Show the full one-time price and exact credit quantity. Explain “1 ภาพหรือ 1 หน้า PDF = 1 เครดิต; ไฟล์หลายหน้าใช้ตามจำนวนหน้า” beside the packs and before starting a scan. State that unused credits roll over monthly and do not expire. Do not show the lower-priced example top-ups as concurrent offers.
- Until OCR passes its accuracy and capacity gates, show the scan packs as unavailable for purchase with an explicit reason; do not accept payment for unusable credits.
- Show current available, reserved, and used credits on the Organization billing page, plus purchase history. Owner initiates a one-time Stripe checkout. Return from checkout shows pending verification until a verified local purchase grants credits; repeat callbacks never add credits twice.
- State that purchased credits do not raise Document allowance. Show the page count and credits required before an eligible scan begins. When balance is insufficient, retain the Document, show missing credits, and allow the user to select pending Documents after a purchase. Explain that failures release reservations, system retries and duplicates cost no extra credit, and unused credits remain after a paid Plan ends.

## Payment and failure states

| State | Message and action |
|---|---|
| Checkout pending | “กำลังตรวจสอบการชำระเงิน” with a refresh/retry path; retain current Entitlements. |
| Active | Current Plan, full next charge, renewal date, and interval. |
| Scheduled change | Current and next Plan/interval, effective date, and any amount due then. |
| Scheduled cancellation | Paid-through date and undo action. |
| Past due, within 7-day grace | Failure date, grace deadline, current retained access, and Owner action to update card/pay the outstanding invoice. Use text and icon with the warning color. |
| Grace expired | Explain Free/Plan Over Limit restrictions and that Documents remain; show the outstanding-payment amount and 30-day recovery deadline, with any unexpired paid Document carry suspended until verified recovery. After the deadline, offer new Checkout rather than implying the old Subscription can resume. |
| Provider temporarily unavailable | Preserve and show the last verified local state with a neutral retry message; never imply a charge or Plan change succeeded. |

## Review checks

- Verify at 375, 768, 1024, and 1440 px, keyboard and screen reader order, 44 px touch targets, visible focus, and no horizontal overflow. Keep status meaning in text as well as color.
- Test the exact amounts, dates, tax wording, Trial transition, downgrade, failed renewal, grace expiry, and cancellation confirmation with representative Thai text before release.
