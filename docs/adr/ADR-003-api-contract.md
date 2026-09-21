# ADR-003: API Contract

## Status

Accepted

## Context

ZERO → AI ENGINEER exposes a REST API through the Go modular monolith. The API must provide a stable versioned boundary while preserving the architecture:

Handler → Service → Domain → Interface → Infrastructure

Authentication is server-managed. Session credentials must not be exposed through URLs or JSON responses, and HTTP handlers must not access PostgreSQL directly.

The existing server provides `GET /health` and `GET /ready`. These endpoints must remain available as operational endpoints while the versioned application API is introduced.

## Decision

### API Base Path

- Versioned application endpoints use the `/api/v1` base path.
- Operational endpoints `/health` and `/ready` remain available at their existing paths.
- API handlers depend on application services and interfaces, never on `database.DB`, `pgxpool.Pool`, or PostgreSQL-specific types.
- HTTP handlers must not contain SQL or persistence logic.

### Authentication Boundary

- Authentication uses server-managed opaque session credentials.
- The raw session token is carried only in the secure session cookie boundary.
- Raw session tokens must never appear in JSON responses, request URLs, logs, or domain models.
- Authentication middleware calls `session.Service.AuthenticateSession`.
- On successful authentication, the authenticated `Session` is stored in request context for downstream handlers.
- Handlers use the authenticated session from context rather than accepting a client-supplied session ID for identity-sensitive operations.

### Logout Endpoint

The first session HTTP endpoint is:

```text
POST /api/v1/auth/logout
```

- Logout uses the authenticated session ID from request context.
- The client must not submit a session ID in the request body, query string, or path.
- The endpoint clears the session cookie after logout.
- Logout is idempotent from the HTTP client's perspective.
- Missing, invalid, expired, and revoked authentication must not be distinguished in the HTTP response.
- Successful logout returns `204 No Content`.

### JSON Errors

Non-success API responses use structured JSON error responses with a stable top-level error code, for example:

```json
{"error":"unauthorized"}
```

Authentication failures use a generic `unauthorized` response and must not disclose whether the credential was missing, invalid, expired, revoked, or unknown.

### Cross-Origin and CSRF Protection

- Cookie-authenticated state-changing requests require Origin validation and CSRF protection appropriate to the request flow.
- CORS uses an explicit configured allowlist.
- Credentialed CORS responses must never use the wildcard origin (`*`).
- The API must not permit arbitrary origins to perform credentialed requests.

## Consequences

- Clients have a stable `/api/v1` namespace for application endpoints.
- Health and readiness probes remain compatible with existing deployment and operational tooling.
- HTTP code remains independent from PostgreSQL infrastructure.
- Authentication policy is centralized in the session service and middleware rather than duplicated in handlers.
- Future authenticated endpoints can consume the request-context session without exposing session credentials.
- CSRF and CORS configuration are part of the security boundary and must be tested as API behavior.
- Registration, login, and session creation are separate application capabilities and are not implied by this ADR.

## Testing Requirements

The API boundary must include tests for:

- versioned route registration
- preservation of `/health` and `/ready`
- cookie authentication middleware behavior
- request-context session propagation
- generic authentication failures
- logout cookie clearing and idempotency
- rejection of client-supplied session identifiers
- Origin/CSRF enforcement
- explicit CORS allowlist behavior
- structured JSON error responses
