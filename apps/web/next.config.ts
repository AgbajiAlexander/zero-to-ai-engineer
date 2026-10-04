import type { NextConfig } from "next";

function apiOrigin(): string {
  const configuredOrigin = process.env.API_ORIGIN;
  const origin = configuredOrigin || (process.env.NODE_ENV === "development" ? "http://localhost:8080" : "");

  if (!origin) {
    throw new Error("API_ORIGIN must be configured for production deployments.");
  }

  let parsed: URL;
  try {
    parsed = new URL(origin);
  } catch {
    throw new Error("API_ORIGIN must be an absolute HTTP(S) origin.");
  }

  if (
    (parsed.protocol !== "http:" && parsed.protocol !== "https:") ||
    parsed.username !== "" ||
    parsed.password !== "" ||
    parsed.pathname !== "/" ||
    parsed.search !== "" ||
    parsed.hash !== ""
  ) {
    throw new Error("API_ORIGIN must be an absolute HTTP(S) origin without a path or credentials.");
  }

  if (process.env.NODE_ENV === "production" && parsed.protocol !== "https:") {
    throw new Error("API_ORIGIN must use HTTPS in production.");
  }

  return parsed.origin;
}

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${apiOrigin()}/api/:path*`,
      },
    ];
  },
};

export default nextConfig;
