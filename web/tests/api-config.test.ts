import assert from "node:assert/strict";
import test from "node:test";
import { getAPIConfig } from "../lib/api-config.ts";

test("staging API requires HTTPS and a server token", () => {
  assert.throws(() => getAPIConfig({ NODE_ENV: "production", API_BASE_URL: "http://api.example", DASHBOARD_API_TOKEN: "token" }));
  assert.deepEqual(
    getAPIConfig({ NODE_ENV: "production", API_BASE_URL: "https://api.example/path", DASHBOARD_API_TOKEN: "token" }),
    { baseURL: "https://api.example", token: "token" },
  );
  assert.equal(getAPIConfig({ NODE_ENV: "production", API_BASE_URL: "http://127.0.0.1:8080", DASHBOARD_API_TOKEN: "token" }).baseURL, "http://127.0.0.1:8080");
});
