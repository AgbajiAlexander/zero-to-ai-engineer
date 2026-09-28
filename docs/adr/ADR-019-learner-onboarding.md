# ADR-019: Learner Onboarding Architecture

## Status

Accepted

## Context and Problem

With the authentication boundary firmly established, users can securely register and log in to the ZERO → AI ENGINEER platform. However, a registered user is not yet a fully configured "Learner."

To begin their educational journey, users must complete an onboarding process that establishes their foundational profile. We need to define the architectural boundary for this learner profile, ensuring it remains server-authoritative, adheres to existing security protocols, integrates seamlessly with existing session authentication, and respects the modular monolith architecture without polluting the existing `auth/user` domain.

## Goals and Non-Goals

### Goals
- Establish the `learner` domain boundary (Handler → Service → Domain → Repository).
- The learner profile must be separate from the auth/user domain.
- Define the minimum required learner profile state for V1 containing only the `display_name`.
- Define a secure API contract for retrieving and initializing the learner profile.
- Guarantee that all authenticated identity comes exclusively from the server-managed session.
- No client-controlled user identifier should be accepted by the API.
- Ensure the learner state remains fully server-authoritative.

### Non-Goals
The following are explicitly excluded from this milestone:
- Separate learner UUID (V1 uses `users.id` foreign key).
- An explicit `onboarding_status` column.
- Diagnostic assessment.
- Skill graph.
- Adaptive roadmap.
- XP, coins, badges, or streaks.
- AI mentor implementation.
- Any changes to existing auth/user contracts.

## Data Model

The database introduces a `learners` table that shares its primary key with the `users` table, establishing a strict one-to-one relationship.

```sql
CREATE TABLE learners (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Onboarding State
Onboarding state is modeled implicitly by the presence of a row in the `learners` table:
- **NOT_STARTED**: No corresponding `learners` row exists.
- **COMPLETED**: A `learners` row exists.

## API Contracts

### Retrieve Learner Profile
```text
GET /api/v1/learners/me
```
- **Description:** Retrieves the authenticated user's learner profile.
- **Response (200 OK):** Returned when the learner profile exists.
- **Response (404 Not Found):** Returned when the learner profile does not exist.

### Initialize Learner Profile
```text
POST /api/v1/learners/onboard
```
- **Description:** Submits the initial onboarding payload to create the learner profile.
- **Request Body:** Requires `display_name`.
- **Response (201 Created):** Returned when the learner profile is successfully created.
- **Response (400 Bad Request):** Returned for validation errors (e.g. display name validation fails).
- **Response (409 Conflict):** Returned when the profile already exists or when there are concurrent creation conflicts. Database-enforced uniqueness/primary-key protection must be mapped to this response.

## Security and Privacy

- **IDOR Prevention:** By ensuring that all operations derive identity strictly from the server-side session context, the API never accepts a client-controlled user identifier. This reduces the Insecure Direct Object Reference (IDOR) attack surface.
- **Authorisation:** The authenticated session sufficiently establishes the user's identity, and operations may only access or create that specific user's own learner profile.

## Validation Requirements

The `Service` layer must enforce reasonable validation on `display_name`:
- Must be required.
- Must trim/normalise whitespace.
- Must have sensible minimum/maximum length constraints.
- Must reject control characters.
- Must preserve legitimate Unicode names.
- Display names are not globally unique.
- No profanity filter is required in V1.

## Testing Requirements

Implementation of this milestone must include:
- Service unit tests.
- PostgreSQL integration tests.
- Concurrent onboarding test to ensure database-level constraints prevent duplicate profiles.
- HTTP tests for 200/201/400/404/409 responses.
- Rejection of unauthenticated requests.
- Conflict mapping for database constraint violations.
