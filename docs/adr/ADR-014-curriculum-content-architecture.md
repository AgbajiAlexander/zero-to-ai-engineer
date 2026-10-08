# ADR-014: Curriculum Content Architecture

## Status

Accepted

## Context

The platform's curriculum is intended to guide learners from computational
thinking through programming, data, machine learning, and production AI. The
README describes the curriculum as a skill graph with prerequisites,
milestones, missions, assessments, and projects.

The Go modular monolith and PostgreSQL remain the application and learner-state
authorities. Curriculum content must be reviewable, version-controlled, and
separate from a learner's progress. Its model must support future missions,
evidence, assessment, and adaptive progression without defining those systems
prematurely.

ADR-019 deliberately excludes a skill graph, diagnostic assessment, and
adaptive roadmap from learner onboarding. ADR-015 and ADR-016 are identified by
`AGENTS.md` as the future authorities for evidence/evaluation and progression,
respectively, but are not yet present in the repository. This proposal
therefore defines curriculum content and structure only.

## Goals

- Define a versioned, reviewable source of truth for curriculum content.
- Represent skills and prerequisite relationships in a form that can be
  validated deterministically.
- Connect learning objectives and milestones to future learning activities
  without embedding learner state in content.
- Keep content delivery within the existing Go API modular monolith.
- Ensure content can be corrected and reviewed without silently rewriting
  historical learner evidence or progress.

## Non-Goals

- Defining assessment rubrics, evidence schemas, grading, or evaluation policy;
  these require ADR-015.
- Defining mastery thresholds, learner progression state, roadmap decisions,
  or mission assignment; these require ADR-016.
- Defining diagnostic assessment or changing learner onboarding in ADR-019.
- Executing learner-provided code in the API or using curriculum content as
  executable instructions.
- Introducing an AI authoring or AI grading system.
- Defining the sandbox execution model, which remains governed by ADR-008.

## Decision

### Curriculum as version-controlled content

- Keep canonical curriculum source files in the repository under
  `apps/api/content/curriculum/`, alongside the Go package that embeds and
  validates them.
- Use versioned JSON documents with a documented schema and Markdown strings
  for authored instructional prose. Use Go's standard JSON support for
  decoding; do not add a content-format dependency for the initial version.
- Each document contains `schema_version`, `version`, `title`, `skills`,
  `milestones`, and `activities`. A skill has an `id`, `title`, one or more
  measurable `objectives`, and optional prerequisite skill IDs. A milestone
  has an `id`, `title`, and one or more skill IDs. An activity reference has
  an `id`, a `kind` (`mission`, `project`, or `assessment`), and one or more
  skill IDs. IDs are unique within each collection; every skill belongs to
  exactly one milestone.
- Treat PostgreSQL as the authority for learner and application state, not as
  the initial authoring store for curriculum documents.
- Publish content only through reviewed source-control changes and a normal
  application deployment. Do not provide runtime content-editing endpoints in
  the initial implementation.
- Include a stable content version identifier in every published curriculum
  document. Do not silently change the meaning of a published identifier.

### Content model and graph

- A curriculum version contains stable identifiers and display metadata for
  skills, milestones, and learning-activity references.
- Skills state observable learning objectives and may declare prerequisite
  skill identifiers.
- Prerequisites form a directed acyclic graph within a curriculum version.
  Validation rejects missing references, duplicate identifiers, and cycles.
- Milestones group related skills for navigation and orientation; they do not
  declare mastery or completion.
- Learning-activity references connect curriculum objectives to future
  missions, projects, or assessment specifications by stable identifier.
  Their execution, submission, evaluation, and learner assignment semantics
  are outside this ADR.
- Curriculum content contains no learner progress, assessment result, mastery,
  confidence, reward, or roadmap fields.

### Version lifecycle and compatibility

- Content revisions are traceable through source control and carry an explicit
  curriculum version identifier.
- A published version is immutable in meaning. Corrections that change
  objectives, prerequisites, or activity requirements produce a new version.
- A previously published version remains available in source control for
  reproducibility. Runtime retention, learner-version pinning, and retirement
  policy must be decided alongside ADR-015/ADR-016 before assignment workflows
  are implemented.
- Stable identifiers may be referenced by external content only when they
  resolve within the declared curriculum version.

### Validation and delivery

- Curriculum loading and validation belong to a curriculum package in the Go
  modular monolith, separate from HTTP handlers and learner persistence.
- Validate curriculum documents during tests and application build/release.
  Invalid JSON, unsupported schema versions, duplicate IDs, unresolved
  references, and cyclic prerequisites must fail validation explicitly.
- Expose curriculum through authenticated, read-only API contracts under
  `/api/v1`, consistent with ADR-003. Both routes require a valid
  server-managed session:
  - `GET /api/v1/curriculum` returns the current published curriculum.
  - `GET /api/v1/curriculum/{version}` returns that exact published version
    or `404 Not Found` when it is unavailable.
- Return the curriculum document as JSON. The current route uses
  `Cache-Control: private, max-age=300`; a version-pinned successful response
  uses `Cache-Control: private, max-age=300, immutable`. Do not cache
  unauthorized responses or share authenticated content through a public
  cache.
- The frontend renders content as content only. Markdown or other authored
  text must not be interpreted as trusted HTML or executable code. Any future
  rich-content renderer requires an explicit, tested sanitization policy.

### Content quality and safety

- Each skill must have a stable ID, a concise title, and measurable learning
  objectives.
- Authored examples and activities must be reviewed by a human before
  publication; AI-generated material is not trusted or published
  automatically.
- Avoid personal data and secrets in curriculum documents.
- Code snippets are inert instructional text. They must never be evaluated or
  executed inside the normal API process.
- Content changes that affect activity requirements must identify the related
  mission/assessment owners and preserve compatibility with the applicable
  future ADRs.

## Testing Requirements

- Schema decoding and validation tests cover valid and invalid content.
- Graph tests cover valid prerequisite order, missing nodes, duplicate IDs,
  self-dependencies, and cycles.
- Version tests ensure references are scoped to a version and published
  versions are not silently replaced.
- API tests verify read-only behavior, response contracts, authentication,
  version lookup, cache policy, and safe error handling.
- Frontend tests verify authored content is rendered as inert text/approved
  markup and not as executable HTML.

## Consequences

- Curriculum changes are reviewable in Git and reproducible from a deployed
  source revision.
- Content validation is deterministic and does not depend on AI output.
- Learner progress and curriculum definition remain separate concerns.
- The initial delivery design avoids new persistence tables and dependencies
  for content authoring.
- Assignment, content retention, assessment, evidence, and progression remain
  intentionally unresolved until their relevant architecture decisions are
  accepted.

## Deferred Decisions

- What retention and learner-version pinning guarantees are required when a
  learner has started a curriculum version?
- Should editorial workflow remain Git-based after the first release, or is a
  separate authoring requirement expected?

Resolve these questions before adding learner-version pinning or runtime
editorial workflows.

## Related ADRs

- ADR-002: Database Architecture
- ADR-003: API Contract
- ADR-008: Learner Code Execution & Secure Sandbox Architecture
- ADR-015: Learning Evidence, Assessment & Evaluation Architecture
  (referenced by `AGENTS.md`; not yet present)
- ADR-016: Learner Progression, Mastery & Adaptive Decision Engine
  (referenced by `AGENTS.md`; not yet present)
- ADR-019: Learner Onboarding Architecture
