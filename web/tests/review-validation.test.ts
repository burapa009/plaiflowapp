import assert from "node:assert/strict";
import test from "node:test";
import { validateReviewField } from "../app/o/[organization]/documents/[document]/review-validation.ts";
test("unknown optional values stay empty; required values cannot be verified empty", () => {
 assert.equal(validateReviewField("seller_tax_id", "", false), "");
 assert.ok(validateReviewField("total_amount", "", true));
});
test("reject invalid leap dates, truncated tax IDs, guessed currencies and malformed amounts", () => {
 assert.ok(validateReviewField("issue_date", "2026-02-29", false));
 assert.equal(validateReviewField("issue_date", "2024-02-29", false), "");
 assert.ok(validateReviewField("seller_tax_id", "01234567890", false));
 assert.equal(validateReviewField("seller_tax_id", "0123456789012", false), "");
 assert.ok(validateReviewField("currency", "บาท", false));
 assert.ok(validateReviewField("total_amount", "1,000.00", false));
 assert.ok(validateReviewField("total_amount", "1000", false));
 assert.equal(validateReviewField("total_amount", "1000.00", false), "");
});
