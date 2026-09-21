# ADR-018: Identity & Password Authentication

## Status

Accepted

## Context

ZERO → AI ENGINEER has completed the opaque session authentication boundary. The project now needs identity operations for registration, password authentication, and retrieval of the currently authenticated user.

The existing architecture is:

Handler → Identity/Auth Service → User Repository → PostgreSQL

The existing session architecture remains:

Identity/Auth Service → Session Service → Session Repository → PostgreSQL

Password credentials and session credentials remain separate security boundaries. Passwords are handled by the existing Argon2id password hashing and verification implementation. Session credentials are handled by the existing opaque-token session service.

## Decision

### Registration

The API exposes:

```text
POST /api/v1/auth/register
```

Request:

```json
{
  "email": "learner@example.com",
  "password": "..."
}
```

Registration must:

- validate and normalize the email address
- enforce a password length of 12–128 characters
- hash the password using the existing Argon2id password boundary
- create a user with an application-generated UUID
- use the database/application default learner role
- create the account active by default
- never persist the raw password
- never return a password or password hash
- never log credentials

A successful registration returns:

```text
201 Created
```

with a safe user representation only:

```json
{
  "user": {
    "id": "...",
    "email": "learner@example.com",
    "role": "LEARNER"
  }
}
```

Registration does not create a session and does not issue a session cookie.

Duplicate normalized email addresses return:

```text
409 Conflict
```

The response must not include password data or database details.

### Email Verification

Email verification is deferred.

- `email_verified_at` remains available for a future verification flow.
- Registration does not issue verification credentials in this milestone.
- Unverified accounts may log in.
- A future email-verification decision may change the login policy through an ADR amendment.

### Login

The API exposes:

```text
POST /api/v1/auth/login
```

Request:

```json
{
  "email": "learner@example.com",
  "password": "..."
}
```

Login must:

1. normalize the email
2. retrieve the user through the user repository
3. verify the password with the existing password-verification boundary
4. reject inactive or deleted accounts
5. generate a new application-side session UUID
6. call the existing `session.Service.CreateSession`
7. issue the returned raw token only as the secure session cookie

The session lifetime is 24 hours.

The existing `session.Service` contract remains unchanged. Login supplies:

```go
session.CreateSessionInput{
    SessionID: generatedSessionID,
    UserID:    user.ID,
    ExpiresAt: now.Add(24 * time.Hour),
}
```

The raw token returned by the session service:

- remains outside the domain `Session`
- is never sent to the repository
- is never returned as JSON
- is never placed in a URL
- is never logged
- is placed only in the `__Host-session` cookie

A successful login returns:

```text
200 OK
```

with the safe user representation and the secure session cookie.

Login failures for unknown email, incorrect password, inactive account, and deleted account are intentionally indistinguishable:

```text
401 Unauthorized
```

```json
{
  "error": "unauthorized"
}
```

No failure response may reveal whether the email exists or why authentication failed.

### Session Fixation

Every successful login creates a new session. Existing client cookies must not be adopted as an authenticated session. The login flow must not reuse a caller-supplied session ID or token.

### Current User

The API exposes:

```text
GET /api/v1/me
```

The endpoint obtains identity from the authenticated `session.Session` stored in request context by existing authentication middleware. It must not accept a client-supplied user ID.

A successful response is:

```text
200 OK
```

```json
{
  "user": {
    "id": "...",
    "email": "learner@example.com",
    "role": "LEARNER"
  }
}
```

Password hashes, session tokens, token hashes, deletion fields, and unnecessary internal account data are excluded.

Missing, invalid, expired, revoked, or unknown authentication returns the existing generic `401 Unauthorized` response.

### User Repository Boundary

The identity layer depends on a PostgreSQL-independent user repository interface. The minimum operations are:

```go
type UserRepository interface {
    CreateUser(ctx context.Context, user User) error
    FindUserByEmail(ctx context.Context, email string) (User, error)
    FindUserByID(ctx context.Context, id string) (User, error)
}
```

The repository owns persistence and database-boundary UUID validation. The identity service owns email normalization, password policy, password verification, account-state policy, and session-creation orchestration.

HTTP handlers must not access PostgreSQL, `database.DB`, `pgxpool.Pool`, or SQL.

### Application-Generated UUIDs

- User and session UUIDs are generated by the Go application layer.
- UUID generation must use a cryptographically secure UUID implementation.
- PostgreSQL continues to store native UUID values.
- PostgreSQL must not generate application identity UUIDs.
- Registration and login must not accept client-generated identity UUIDs.

The existing `session.Service` remains unchanged; the identity layer supplies the generated session ID through `CreateSessionInput`.

### Rate Limiting Boundary

Login and registration require rate-limiting boundaries before production exposure.

The identity layer may depend on an interface such as:

```go
type RateLimiter interface {
    Allow(ctx context.Context, key string) error
}
```

Rate limiting must not be implemented through SQL in HTTP handlers or user repositories. The limiter key and implementation must not log passwords, tokens, or password hashes.

A process-local limiter is not considered sufficient for a multi-instance deployment. The deployment model must determine whether a durable/distributed limiter is required.

### Credential and Protocol Scope

- JWTs are not used.
- OAuth and social login are not introduced.
- No refresh-token system is introduced.
- Session authentication remains opaque and cookie-only.
- The existing `POST /api/v1/auth/logout` endpoint remains unchanged.
- CSRF, Origin, CORS, and secure-cookie rules from ADR-003 and ADR-007 continue to apply.

## Consequences

- Registration creates an account but does not authenticate the caller.
- Login creates a fresh 24-hour server-managed session.
- Unverified accounts can authenticate until a future verification policy changes that decision.
- Inactive and deleted accounts cannot authenticate.
- Generic login errors prevent account enumeration through the login endpoint.
- Duplicate registration currently returns `409 Conflict`, which may reveal account existence; this is an accepted V1 trade-off.
- The users table is sufficient for this milestone without a migration.
- Email delivery, verification tokens, password reset, MFA, social login, refresh tokens, and advanced distributed rate limiting remain future work.

## API Error Contract

Non-success responses use structured JSON errors.

Authentication failures:

```text
401 Unauthorized
{"error":"unauthorized"}
```

Duplicate email:

```text
409 Conflict
{"error":"email_already_registered"}
```

Malformed request or password-policy failure:

```text
400 Bad Request
{"error":"invalid_request"}
```

Unexpected internal failures return a generic internal error without database, SQL, credential, or stack-trace details.

## Security Requirements

Implementations must test and enforce:

- password length boundaries at 12 and 128 characters
- password rejection outside the allowed range
- Argon2id hashing and verification
- no raw password or password hash in API responses or logs
- generic login failures
- inactive/deleted-account rejection
- fresh session creation on every login
- 24-hour session expiration
- secure `__Host-session` cookie attributes
- raw-token absence from JSON, URLs, logs, and domain models
- authenticated-context identity for `/me`
- rejection of client-supplied user identity
- registration and login rate-limiting boundaries
- existing CSRF, Origin, and explicit CORS rules

## Related ADRs

- ADR-002: Database Architecture
- ADR-003: API Contract
- ADR-007: Authentication, Authorisation & Identity Security
