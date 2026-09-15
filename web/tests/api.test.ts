import assert from "node:assert/strict";
import test from "node:test";
import { getSnapshot } from "../lib/api.ts";

test("dashboard request carries the server token and a request ID", async () => {
  const originalFetch = globalThis.fetch;
  let headers = new Headers();
  process.env.API_BASE_URL = "https://api.staging.example";
  process.env.DASHBOARD_API_TOKEN = "test-token";
  globalThis.fetch = async (_input, init) => {
    headers = new Headers(init?.headers);
    return new Response(JSON.stringify({ counts: { received: 0, processed: 0, ignored: 0, retryable: 0, failed: 0 }, api: "ok", database: "ok", worker: "ok" }));
  };

  try {
    await getSnapshot();
    assert.equal(headers.get("Authorization"), "Bearer test-token");
    assert.match(headers.get("X-Request-ID") ?? "", /^[0-9a-f-]{36}$/);
  } finally {
    globalThis.fetch = originalFetch;
    delete process.env.API_BASE_URL;
    delete process.env.DASHBOARD_API_TOKEN;
  }
});
