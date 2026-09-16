import type { NextConfig } from "next";
import { getAPIBaseURL } from "./lib/api-config.ts";

const nextConfig: NextConfig = {
  agentRules: false,
  poweredByHeader: false,
  experimental: { serverActions: { bodySizeLimit: "11mb" } },
  async rewrites() {
    const api = getAPIBaseURL(process.env);
    return [
      { source: "/api/auth/:path*", destination: `${api}/v1/auth/:path*` },
      { source: "/api/drive/callback", destination: `${api}/v1/drive/callback` },
      { source: "/api/o/:organization/drive/:action", destination: `${api}/v1/o/:organization/drive/:action` },
      { source: "/api/o/:organization/vendors/export.csv", destination: `${api}/v1/o/:organization/vendors/export.csv` },
      { source: "/api/o/:organization/vendors/export.xlsx", destination: `${api}/v1/o/:organization/vendors/export.xlsx` },
    ];
  },
};

export default nextConfig;
