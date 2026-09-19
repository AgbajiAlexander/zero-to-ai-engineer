# ZERO → AI ENGINEER - Agent Rules

This file establishes the engineering rules and architectural guidelines for the "ZERO → AI ENGINEER" platform.
As an AI coding implementation agent, you must follow the architecture provided and must NOT redesign, replace, simplify, or introduce alternative architecture unless explicitly asked.

## Engineering Rules

1. Follow the approved architecture and ADRs.
2. Do not introduce microservices. The backend is a Go modular monolith unless a future ADR explicitly changes this.
3. The frontend is Next.js + React + TypeScript.
4. The backend is Go.
5. PostgreSQL is the authoritative database.
6. Gemini is accessed only through a server-side AI Gateway.
7. Never expose API keys, database credentials, session secrets, or other secrets to the frontend.
8. The frontend must never directly modify authoritative learner state.
9. Mastery, confidence, XP, coins, badges, streaks, roles, permissions, curriculum state, and roadmap state are server-authoritative.
10. AI may assist, explain, evaluate, recommend, and coach, but AI must not directly mutate authoritative learner state.
11. Learner code must never execute inside the normal Go API process.
12. Learner code execution must use an isolated sandbox architecture.
13. All database schema changes must use version-controlled migrations.
14. Important state transitions must be auditable.
15. Authentication and authorisation must be implemented securely.
16. Resource ownership checks must prevent IDOR vulnerabilities.
17. Use parameterised SQL or safe database abstractions.
18. Validate all untrusted input.
19. Use secure server-managed sessions.
20. Use Argon2id for password hashing.
21. Use secure cookies where cookie-based sessions are used.
22. Implement appropriate CSRF, CORS, rate limiting, and security headers.
23. Never weaken security merely to make tests pass.
24. Every feature must include appropriate tests.
25. AI-generated code is treated exactly like human-written code and must pass architecture, security, quality, and testing requirements.
26. Do not silently modify the database schema.
27. Do not remove or weaken failing tests without an explicit engineering reason.
28. Do not add dependencies unless there is a clear justification.
29. Prefer simple, maintainable solutions over unnecessary complexity.
30. Keep the modular monolith modular through clear package boundaries.
31. Keep business logic out of HTTP handlers where practical.
32. Use the architectural flow:
    Handler → Service → Domain → Interface → Infrastructure.
33. Use structured logging and request IDs.
34. Never log passwords, session tokens, API keys, database credentials, or unnecessary sensitive learner data.
35. Treat all learner-provided content as untrusted input, including content sent to the AI Mentor.
36. Defend against prompt injection and AI data leakage.
37. Use structured AI outputs where appropriate and validate them before use.
38. AI failures must not corrupt learner progress or authoritative state.
39. Deterministic application logic remains authoritative over AI recommendations.
40. Do not execute arbitrary shell commands, SQL, filesystem operations, or network operations based solely on AI-generated instructions.
41. Keep development, staging, and production environments separated.
42. Never commit secrets to Git.
43. Keep documentation close to the implementation and update it when architecture changes.
44. Follow the Definition of Done for every feature:
    - implementation
    - validation
    - authorisation
    - business rules
    - tests
    - security
    - observability
    - documentation
    - successful build

## Development Workflow

- Work incrementally.
- Explain important implementation decisions.
- Before creating multiple files, state which files you intend to create and why.
- Do not generate the entire application in one step.
- Do not create placeholder architecture that bypasses the real design.
- Prefer small verified changes.
- After each meaningful change, run the appropriate formatter, compiler, linter, and tests.
- If something fails, investigate the root cause rather than bypassing the failure.
- Ask for clarification only when an architectural decision is genuinely missing.
- Otherwise follow the approved ADRs.

## Current Architecture

Frontend:
Next.js + React + TypeScript

Backend:
Go modular monolith

Database:
PostgreSQL

AI:
Server-side AI Gateway → Gemini

Deployment:
GitHub + CI/CD + Docker + Vercel/frontend hosting + managed backend/database infrastructure

## Implementation Phases

Repository foundation
→ Go API foundation
→ PostgreSQL
→ Docker
→ Authentication
→ Learner onboarding
→ Curriculum
→ Missions
→ Assessment
→ Evidence
→ Mastery
→ Adaptive roadmap
→ AI Mentor
→ Sandbox
→ Collaboration
→ Projects
→ Production hardening

## Architectural Authority

The following Architecture Decision Records are authoritative project documents:

ADR-001 — Production Architecture
ADR-002 — Database Architecture
ADR-003 — API Contract
ADR-004 — Engineering Repository & Code Architecture
ADR-005 — Mastery, Assessment & Adaptive Roadmap Engine
ADR-006 — AI Safety, Prompt Security & AI Governance
ADR-007 — Authentication, Authorisation & Identity Security
ADR-008 — Learner Code Execution & Secure Sandbox Architecture
ADR-009 — Observability, Auditability, Reliability & Incident Response
ADR-010 — Data Privacy, Governance & Responsible AI
ADR-011 — Testing Strategy & Quality Engineering
ADR-012 — Deployment, Infrastructure & DevOps Architecture
ADR-013 — Frontend Architecture & UX System
ADR-014 — Curriculum Content Architecture
ADR-015 — Learning Evidence, Assessment & Evaluation Architecture
ADR-016 — Learner Progression, Mastery & Adaptive Decision Engine
ADR-017 — AI Mentor & Personal Learning Assistant Architecture

When implementing a feature, identify the relevant ADRs before making architectural decisions.

If implementation requirements appear to conflict with an ADR, stop and report the conflict rather than silently changing the architecture.

New architectural decisions must be documented in a new ADR or an explicitly approved amendment to an existing ADR.
