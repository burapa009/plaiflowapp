export function getAPIBaseURL(env: NodeJS.ProcessEnv) {
  const baseURL = env.API_BASE_URL;
  if (!baseURL) throw new Error("Server API configuration is missing");
  const url = new URL(baseURL);
  const local = url.hostname === "localhost" || url.hostname === "127.0.0.1";
  if (!local && url.protocol !== "https:") {
    throw new Error("Deployed API_BASE_URL must use HTTPS");
  }
  return url.origin;
}

export function getAPIConfig(env: NodeJS.ProcessEnv) {
  const token = env.DASHBOARD_API_TOKEN;
  if (!token) throw new Error("Server API configuration is missing");
  return { baseURL: getAPIBaseURL(env), token };
}
