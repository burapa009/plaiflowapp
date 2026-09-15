import { getAPIConfig } from "./api-config";

export type Snapshot = {
  counts: { received: number; processed: number; ignored: number; retryable: number; failed: number };
  api: string;
  database: string;
  worker: string;
};

export async function getSnapshot(): Promise<Snapshot> {
  const { baseURL, token } = getAPIConfig(process.env);
  const response = await fetch(`${baseURL}/v1/dashboard`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
    signal: AbortSignal.timeout(5000),
  });
  if (!response.ok) throw new Error(`Dashboard API returned ${response.status}`);
  return response.json() as Promise<Snapshot>;
}
