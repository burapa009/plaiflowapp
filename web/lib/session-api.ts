import "server-only";

import { getAPIBaseURL } from "./api-config.ts";
import { cookies, headers } from "next/headers";

const csrfCookieName = "__Host-plaiflow-csrf";

export async function sessionGET(path: string) {
  try {
    return await fetch(`${getAPIBaseURL(process.env)}${path}`, {
      headers: { Cookie: (await cookies()).toString() },
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
  } catch {
    return null;
  }
}

export async function sessionPOST(path: string, body: URLSearchParams, redirectMode: RequestRedirect = "follow") {
  const cookieStore = await cookies();
  const csrf = cookieStore.get(csrfCookieName)?.value;
  const origin = (await headers()).get("origin");
  if (!csrf || !origin) return null;
  try {
    return await fetch(`${getAPIBaseURL(process.env)}${path}`, {
      method: "POST",
      headers: {
        Cookie: cookieStore.toString(),
        "Content-Type": "application/x-www-form-urlencoded",
        Origin: origin,
        "X-CSRF-Token": csrf,
      },
      body,
      redirect: redirectMode,
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
  } catch {
    return null;
  }
}

export async function sessionMultipartPOST(path: string, body: FormData) {
  const cookieStore = await cookies();
  const csrf = cookieStore.get(csrfCookieName)?.value;
  const origin = (await headers()).get("origin");
  if (!csrf || !origin) return null;
  try {
    return await fetch(`${getAPIBaseURL(process.env)}${path}`, {
      method: "POST",
      headers: { Cookie: cookieStore.toString(), Origin: origin, "X-CSRF-Token": csrf },
      body,
      cache: "no-store",
      signal: AbortSignal.timeout(10000),
    });
  } catch {
    return null;
  }
}
