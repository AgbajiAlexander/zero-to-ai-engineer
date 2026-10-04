# ADR-020: Initial Hosted Deployment Topology

## Status

Accepted

## Context

The repository defines a Next.js frontend, a Go modular-monolith API, and PostgreSQL as the authoritative database. The first hosted deployment needs to preserve the Go API boundary, PostgreSQL migrations, secure server-managed sessions, and existing browser security rules.

## Decision

- Deploy the Next.js application from `apps/web` to Vercel under the project name `zero-to-ai-engineer`.
- Deploy the Go API as a Dockerized service on Railway in a separately named project, `zero-to-ai-engineer`.
- Use a new Supabase-managed PostgreSQL project named `zero-to-ai-engineer`, leaving unrelated provider projects untouched.
- Keep browser API requests on the Vercel origin. Next.js rewrites `/api/*` to the Railway API using a server-only `API_ORIGIN`, so the browser stores the secure, host-only session cookie on the Vercel origin.
- Configure the Go API's `WEB_ORIGIN` to the exact Vercel deployment origin. Keep the existing secure-cookie, Origin validation, and credentialed CORS requirements.
- Store `DATABASE_URL` and `SESSION_SECRET` only in Railway's server-side environment. Store only `API_ORIGIN` in the Vercel server environment; do not expose database credentials or session secrets to the browser.
- Apply the existing version-controlled `apps/api/migrations` using the migration runner before directing application traffic to the API.
- Use the Supabase project region `eu-north-1` for the initial deployment.

## Consequences

- The browser makes same-origin requests for session-authenticated API operations; the API remains a distinct Go service.
- The Vercel deployment is not functional until the Railway API is reachable and its database migrations have completed.
- Production environment variables and database access must be configured independently for each provider.
- Database schema changes continue to be managed by the existing version-controlled Go migration files; API startup does not run migrations.
- Provider deployment configuration and credentials must not be committed to the repository.

## Non-Goals

- Moving the Go API into Vercel functions.
- Replacing PostgreSQL or the existing migration system.
- Adding AI, curriculum, assessment, or learner progression features.
- Modifying or restoring pre-existing Supabase or Railway projects.

## Related ADRs

- ADR-002: Database Architecture
- ADR-003: API Contract
- ADR-007: Authentication, Authorisation & Identity Security
- ADR-018: Identity & Password Authentication
- ADR-019: Learner Onboarding Architecture
