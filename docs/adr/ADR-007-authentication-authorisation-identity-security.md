# ADR-007: Authentication, Authorisation & Identity Security

## Status

Accepted

## Context

ZERO → AI ENGINEER requires secure server-managed authentication for the Go modular monolith. Password credentials and session credentials have different security properties and must use separate boundaries.

The session model stores a token hash rather than a raw token. The session service owns authentication policy, while the session repository owns persistence. HTTP handlers must remain above the service layer and must not access PostgreSQL directly.

The project must prevent credential disclosure through API responses, URLs, logs, domain objects, and client-controlled identity parameters.

## Decision

### Session Authentication

- Authentication uses opaque, cryptographically secure session tokens.
- Tokens are generated with the existing secure token boundary.
- The raw token is hashed before persistence and lookup.
- The database stores only the token hash.
- The raw token must never be stored in the `Session` domain model.
- The raw token must never appear in JSON, URLs, logs, SQL values, or error messages.
- JWTs are not used for this session boundary.

### Session Cookie

The server-managed session cookie is:

```text
Name: __Host-session
HttpOnly: true
Secure: true
SameSite: Lax
Path: /
Domain: omitted
```

The `__Host-` prefix requires `Secure`, `Path=/`, and no `Domain` attribute. The cookie is the only HTTP transport for the raw session token.

Raw session tokens must not be accepted through query parameters, URL path segments, JSON fields, or authorization URLs.

### Authentication Middleware

- Authentication middleware reads the session cookie.
- It passes the raw token to `session.Service.AuthenticateSession`.
- It does not query PostgreSQL directly.
- It stores the authenticated `Session` in request context.
- Downstream handlers use the context session for identity and ownership checks.
- Handlers must not trust client-supplied session IDs or user IDs for the current authenticated identity.

### Authentication Validity

A session is authentication-valid only when:

```text
session exists
AND session.RevokedAt == nil
AND session.ExpiresAt is after the current time
```

The service rejects empty tokens, revoked sessions, and expired sessions. Authentication lookup does not mutate the session or update `LastSeenAt`.

Missing, invalid, unknown, expired, and revoked credentials must not be distinguished in externally visible authentication responses.

### Logout

The initial HTTP session operation is:

```text
POST /api/v1/auth/logout
```

- Logout obtains the session ID from authenticated request context.
- The client cannot choose the session ID to revoke.
- The service delegates revocation to the existing session repository contract.
- The HTTP operation is idempotent from the client's perspective.
- The response clears the session cookie.
- Logout does not return the raw token, token hash, session ID, or user ID in JSON.

### Password Security

- Passwords use the existing Argon2id password hashing boundary.
- Password hashing is not used for session tokens.
- Password hashes and authentication credentials must never be logged.
- Registration and login are separate capabilities and are not introduced by this ADR's session HTTP boundary.

### CSRF, CORS, and Browser Security

- Cookie-authenticated state-changing requests require Origin validation and CSRF protection.
- CORS uses an explicit allowlist of trusted origins.
- Credentialed CORS responses must never use `Access-Control-Allow-Origin: *`.
- Security headers, restricted methods, and credential handling must be applied consistently at the HTTP boundary.
- Cross-origin requests must not be allowed to perform credentialed state changes without the required CSRF protections.

### Error Handling and Observability

- API errors use structured JSON responses.
- Authentication failures use a generic unauthorized error and do not reveal the underlying condition.
- Internal logs must not include raw tokens, token hashes, password hashes, cookies, or unnecessary identity data.
- Repository errors remain distinguishable inside the service through `errors.Is`, but HTTP responses apply the generic external policy.
- Important authentication and revocation transitions should be auditable without recording secret credentials.

## Consequences

- Session credentials remain opaque to clients and are protected by secure cookie attributes.
- The database remains useful for deterministic token-hash lookup without storing bearer credentials.
- Authentication policy is centralized in the session service.
- HTTP middleware provides one authenticated-session context for handlers.
- Resource ownership and authorisation checks can use the context session while remaining server-authoritative.
- Cookie-authenticated state changes require explicit CSRF and CORS controls.
- A future login flow must create sessions through the session service and return the raw token only through the secure cookie boundary.
- No database, migration, schema, JWT, or HTTP handler changes are implied by this ADR alone.

## Testing Requirements

Authentication and session HTTP implementations must test:

- secure cookie attributes
- absence of raw tokens from JSON, URLs, logs, and domain objects
- valid session authentication
- missing, invalid, expired, revoked, and unknown credentials
- generic external error responses
- request-context session propagation
- logout using the context session ID
- logout idempotency
- CSRF and Origin enforcement
- explicit CORS allowlist behavior
- resource ownership and authorisation checks
- password/session credential boundary separation
