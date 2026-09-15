import assert from "node:assert/strict";
import test from "node:test";
import { taskFormBody, updateTaskBody } from "../lib/task-form.ts";

test("task form trims valid input and rejects invalid dates", () => {
  const valid = new FormData();
  valid.set("title", "  โทรหาลูกค้า  ");
  valid.set("priority", "High");
  valid.set("due_on", "2026-09-30");
  assert.equal(taskFormBody(valid)?.get("title"), "โทรหาลูกค้า");

  const invalid = new FormData();
  invalid.set("title", "งาน");
  invalid.set("due_on", "2026-02-31");
  assert.equal(taskFormBody(invalid), null);
});

test("update form requires a supported status", () => {
  const form = new FormData();
  form.set("title", "งานเดิม");
  form.set("priority", "Normal");
  form.set("status", "Unknown");
  assert.equal(updateTaskBody(form), null);
});
