import assert from "node:assert/strict";
import test from "node:test";
import { formatReviewAmount } from "../app/o/[organization]/documents/[document]/review-amount.ts";

test("review footer formats numeric THB and keeps uncertain OCR text", () => {
  assert.equal(formatReviewAmount("11875.00", "THB"), "฿11,875.00");
  assert.equal(formatReviewAmount("", "THB"), "—");
  assert.equal(formatReviewAmount("อ่านไม่ชัด", "THB"), "อ่านไม่ชัด");
  assert.equal(formatReviewAmount("11,875.00", "THB"), "฿11,875.00");
  assert.equal(formatReviewAmount("11875.00", "USD"), "USD 11,875.00");
});
