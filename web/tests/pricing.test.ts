import assert from "node:assert/strict";
import test from "node:test";
import { formatSatang, intervalLabel } from "../lib/pricing.ts";

test("pricing presents exact prepaid totals for all supported intervals", () => {
  assert.equal(formatSatang(85500), "฿855.00");
  assert.equal(intervalLabel("monthly"), "รายเดือน");
  assert.equal(intervalLabel("six_months"), "6 เดือน");
  assert.equal(intervalLabel("yearly"), "รายปี");
});
