import { getAPIConfig } from "./api-config.ts";
import { randomUUID } from "node:crypto";

export type Snapshot = {
  counts: { received: number; processed: number; ignored: number; retryable: number; failed: number };
  jobs: {
    queued: number; running: number; failed: number; retries: number; stale_reclaims: number;
    oldest_queue_wait_seconds: number; worker_heartbeat_age_seconds: number; export_rows: number; export_bytes: number;
    downloads: number; average_runtime_seconds: number;
  };
  api: string;
  database: string;
  worker: string;
};

export async function getSnapshot(): Promise<Snapshot> {
  const { baseURL, token } = getAPIConfig(process.env);
  const response = await fetch(`${baseURL}/v1/dashboard`, {
    headers: { Authorization: `Bearer ${token}`, "X-Request-ID": randomUUID() },
    cache: "no-store",
    signal: AbortSignal.timeout(5000),
  });
  if (!response.ok) throw new Error(`Dashboard API returned ${response.status}`);
  return response.json() as Promise<Snapshot>;
}
