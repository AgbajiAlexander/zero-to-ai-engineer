import type { NextConfig } from "next";

/**
 * Resolves the Go API origin that Next.js rewrites `/api/*` requests to.
 *
 * Resolution order:
 *  1. API_ORIGIN env var (required on Vercel and CI).
 *  2. Localhost fallback for local development and local production builds.
 *
 * API_ORIGIN is server-only and must never be exposed as NEXT_PUBLIC_*.
 * In production (Vercel / CI) it must be an absolute HTTPS origin.
 */
function apiOrigin(): string {
  const configured = process.env.API_ORIGIN;

  // On Vercel or CI, API_ORIGIN must be explicitly configured.
  const isHosted = Boolean(process.env.VERCEL) || Boolean(process.env.CI);
  if (!configured && isHosted) {
    throw new Error(
      "API_ORIGIN must be set in the Vercel (or CI) environment. " +
        "See apps/web/.env.local.example for local usage.",
    );
  }

  const origin = configured ?? "http://localhost:8080";

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
    throw new Error(
      "API_ORIGIN must be an absolute HTTP(S) origin without a path, credentials, query string, or fragment.",
    );
  }

  // Enforce HTTPS for hosted deployments.
  if (isHosted && parsed.protocol !== "https:") {
    throw new Error("API_ORIGIN must use HTTPS in hosted deployments.");
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
