# ADR-013: Frontend Architecture and UX System

## Status

Accepted

## Context

ZERO → AI ENGINEER uses a Next.js App Router frontend with React and
TypeScript. The frontend communicates with the Go modular-monolith API, which
owns authentication, learner profiles, curriculum state, and all other
authoritative learner data.

The initial web application provides registration, sign-in, and learner
onboarding. The next product phases will add curriculum browsing and learning
workflows. Frontend conventions must support those features without creating a
second source of truth, weakening session security, or coupling presentation
code to backend persistence.

## Goals

- Define a maintainable frontend structure for the existing Next.js application.
- Preserve the API as the authority for identity, authorisation, and learner
  state.
- Keep browser requests same-origin and session credentials inaccessible to
  application JavaScript.
- Establish consistent accessibility, responsive layout, interaction, and
  error-state requirements.
- Allow features to be added incrementally without introducing unnecessary
  dependencies or a second frontend application.

## Non-Goals

- Moving business rules, database access, or authoritative state into Next.js.
- Replacing the Go API with server actions, Next.js route handlers, or Vercel
  functions.
- Selecting a component library or adding a design-system dependency.
- Specifying the curriculum domain, assessment rules, mastery calculation, or
  adaptive roadmap.
- Requiring a wholesale rewrite of the existing registration and onboarding
  page before new frontend work can proceed.

## Decision

### Application and rendering model

- Keep the frontend in `apps/web` using Next.js App Router, React, and
  TypeScript.
- Use React Server Components by default for presentation that does not need
  browser state or event handlers.
- Introduce Client Components only at interactive boundaries such as forms,
  menus, and learner activities that require client-side interaction.
- Keep route-level composition in `app/`; group reusable, feature-specific UI
  and API-facing presentation code by domain as the application grows.
- Do not put domain rules, authorization decisions, or PostgreSQL access in
  frontend components.

### API and session boundary

- The browser calls application endpoints through the same-origin `/api/*`
  rewrite to the Go API.
- `API_ORIGIN` is server-only, required in production, and must be an absolute
  HTTPS origin. It must never use a `NEXT_PUBLIC_*` name.
- The browser may initiate an API operation, but only the Go API may validate,
  authorize, and commit changes to authoritative learner state.
- Session tokens remain in secure, HTTP-only cookies; frontend code must not
  read, store, log, or forward raw session credentials.
- Validate API response shapes at the frontend boundary. Treat server error
  codes as untrusted input and present safe, actionable messages rather than
  raw implementation details.
- Keep API contracts aligned with ADR-003 and the applicable domain ADRs.

### State and forms

- Client state is limited to transient presentation state, such as open panels,
  current form input, pending indicators, and API responses.
- A successful API response may update the displayed view, but does not grant
  the frontend authority to calculate or persist learner progression.
- Forms must use semantic controls, explicit labels, server-compatible
  validation, pending and failure states, and prevention of accidental
  duplicate submission.
- Authentication and other session-bound operations must retain the API's
  Origin/CSRF requirements; frontend code must not work around a rejected
  request by weakening browser or API security.

### UX, accessibility, and styling

- Use semantic HTML and preserve keyboard access, visible focus, meaningful
  accessible names, and status/error announcements.
- Support narrow and wide viewports without hiding essential actions or
  information.
- Respect reduced-motion preferences whenever motion is introduced.
- Maintain a small, documented set of shared visual tokens for colour,
  typography, spacing, borders, and focus treatment. Prefer existing project
  styling and CSS variables before introducing new styling dependencies.
- Provide explicit loading, empty, success, and recoverable error states for
  asynchronous feature flows.

### Testing and quality

- Run ESLint, TypeScript checking, and a production build for frontend changes.
- Add focused tests for non-trivial UI state, input handling, and API response
  interpretation using the repository's established test tooling.
- Verify keyboard and responsive behavior for changed interactive surfaces.
- Test the browser-to-API boundary without exposing secrets or bypassing API
  authorization.

## Consequences

- The frontend remains a presentation and interaction layer over the Go API.
- Client-side JavaScript is scoped to the interactions that need it.
- Features can share UX conventions without introducing a component framework.
- Existing client-rendered flows may be improved incrementally; this decision
  does not require a disruptive migration.
- New shared UI abstractions should be introduced only when more than one
  feature needs them.

## Open Questions for Review

- Should the first shared visual-token set be documented here or in a separate
  design-system specification?
- Which test runner should be adopted when focused component tests become
  necessary? No new test dependency is proposed by this ADR.

## Related ADRs

- ADR-003: API Contract
- ADR-007: Authentication, Authorisation & Identity Security
- ADR-018: Identity & Password Authentication
- ADR-019: Learner Onboarding Architecture
- ADR-020: Initial Hosted Deployment Topology
