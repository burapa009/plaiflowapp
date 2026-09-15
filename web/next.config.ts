import type { NextConfig } from "next";
import { getAPIBaseURL } from "./lib/api-config.ts";

const nextConfig: NextConfig = {
  agentRules: false,
  poweredByHeader: false,
  async rewrites() {
    return [{ source: "/api/auth/:path*", destination: `${getAPIBaseURL(process.env)}/v1/auth/:path*` }];
  },
};

export default nextConfig;
