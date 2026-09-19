# ZERO → AI ENGINEER

ZERO → AI ENGINEER is an adaptive AI apprenticeship platform designed to guide learners from zero programming knowledge to becoming capable AI engineers through evidence-based learning, practical projects, collaboration, assessment, and adaptive progression.

## Vision

To make world-class AI engineering education accessible to anyone, regardless of their starting point, by providing an adaptive AI apprenticeship that guides learners from zero programming knowledge to building real-world, production-ready AI solutions.

## Core Philosophy

The platform is built around these principles:

- Competence over course completion.
- Evidence over passive content consumption or self-claims.
- AI assists; deterministic systems decide.
- Mastery is based on validated evidence.
- Learner independence should increase over time.
- Collaboration and peer learning are part of the learning process.
- Important learner state must be authoritative, auditable, and explainable.
- Security and privacy are built into the architecture.

## Learning Loop

The core learning loop is:

Learn
→ Practise
→ Build
→ Collaborate
→ Explain
→ Demonstrate
→ Receive feedback
→ Improve
→ Audit
→ Adapt
→ Repeat

## Learner Journey

Landing
→ Registration
→ Onboarding
→ Diagnostic Assessment
→ Skill Profile
→ Personalised Roadmap
→ Today's Mission
→ Learn
→ Practise
→ Build
→ Submit Evidence
→ Evaluation
→ Mastery Update
→ XP / Coins / Badges
→ Skill Audit
→ Roadmap Adaptation
→ Next Mission

## Curriculum

The curriculum progresses through:

1. Computational Thinking
2. Programming Foundations / Python
3. Developer Foundations
4. Data Foundations
5. Machine Learning
6. Deep Learning
7. Generative AI
8. AI Engineering
9. Production AI
10. AI Engineer / Capstone

The curriculum is represented as a skill graph with prerequisites, milestones, missions, assessments, and projects.

## Adaptive Learning

The platform does not simply mark lessons as complete.

It collects evidence of what a learner can actually do and uses validated evidence to update learner skill state and determine appropriate next actions.

Possible progression decisions include:

- ADVANCE
- PRACTISE
- REVIEW
- RETRY
- SKIP
- CHALLENGE
- RECOVER
- COLLABORATE
- AUDIT

The authoritative progression system is deterministic and explainable.

## AI Mentor

The AI Mentor is a learning assistant rather than the authority of the platform.

Initial mentor modes include:

- EXPLAIN
- HINT
- SOCRATIC
- DEBUG
- REVIEW
- CHALLENGE
- PROJECT_COACH

The AI Mentor can explain, guide, debug, review, challenge, and coach.

It cannot directly modify:

- mastery
- confidence
- XP
- coins
- badges
- streaks
- roles
- permissions
- curriculum
- roadmap state
- authoritative assessment results

AI is accessed through a server-side AI Gateway.

## Technology Stack

### Frontend

- Next.js
- React
- TypeScript

### Backend

- Go
- Modular monolith architecture
- REST API

### Database

- PostgreSQL

### AI

- Server-side AI Gateway
- Gemini

### Infrastructure

- GitHub
- Docker
- CI/CD
- Vercel for frontend hosting
- Managed backend infrastructure
- Managed PostgreSQL
- Isolated sandbox workers for learner code execution

## Architecture

The high-level architecture is:

Next.js
→ HTTPS
→ Go API
→ PostgreSQL

The Go API contains modular domain services and an AI Gateway.

The AI Gateway communicates with Gemini.

Learner code never executes inside the normal Go API process. Code execution uses an isolated sandbox architecture.

## Backend Architecture

The backend follows:

Handler
→ Service
→ Domain
→ Interface
→ Infrastructure

The backend is a modular monolith.

Microservices are not introduced unless a future Architecture Decision Record explicitly requires them.

## Security

Security principles include:

- Secure server-managed sessions
- Argon2id password hashing
- Authorisation and resource ownership checks
- IDOR protection
- Input validation
- Parameterised SQL
- CSRF protection where applicable
- Restricted CORS
- Rate limiting
- Security headers
- Server-side secrets
- Audit logging
- Least privilege
- AI prompt-injection protection
- AI data minimisation
- Isolated learner-code execution
- No production credentials in sandbox environments

## Testing

Testing is part of every feature.

The project uses:

- Unit tests
- Integration tests
- API/contract tests
- Database tests
- Security tests
- AI evaluation tests
- Sandbox tests
- End-to-end tests
- Reliability tests
- Performance tests
- Regression tests

Go quality checks include:

- gofmt
- go vet
- go test ./...
- go test -race ./...

## Repository Structure

The planned repository structure is:

zero-to-ai-engineer/
├── apps/
│   ├── web/
│   └── api/
├── packages/
│   ├── shared-types/
│   └── validation/
├── docs/
├── scripts/
├── tests/
├── .github/
├── AGENTS.md
├── README.md
├── CONTRIBUTING.md
├── SECURITY.md
├── .env.example
├── .gitignore
├── docker-compose.yml
└── package.json

## Development Philosophy

The project is built incrementally.

Each feature should be:

1. Designed
2. Implemented
3. Tested
4. Security-reviewed
5. Observed
6. Documented
7. Verified
8. Committed

AI coding agents may accelerate implementation, but they do not replace engineering judgement, architecture, testing, security review, or human ownership.

## Architecture Decision Records

The architecture is documented through Architecture Decision Records covering:

- Production Architecture
- Database Architecture
- API Contract
- Repository and Code Architecture
- Mastery and Adaptive Learning
- AI Safety and Governance
- Authentication and Identity Security
- Secure Code Execution
- Observability and Reliability
- Privacy and Responsible AI
- Testing and Quality Engineering
- Deployment and DevOps
- Frontend Architecture
- Curriculum Architecture
- Learning Evidence and Evaluation
- Learner Progression and Adaptive Decisions
- AI Mentor Architecture

See the project's architecture documentation for the authoritative decisions.

## Development Workflow

The project should be developed in small, verified increments.

Do not generate the entire application at once.

After each meaningful change:

1. Inspect the change.
2. Format the code.
3. Run appropriate tests.
4. Run static checks.
5. Review security implications.
6. Confirm the architecture has not been violated.
7. Commit the change when verified.

## Project Status

Current stage:

Architecture completed through ADR-017.

Current implementation stage:

Repository foundation.

The next implementation milestones are:

1. Repository foundation
2. Go API foundation
3. PostgreSQL and migrations
4. Docker development environment
5. Authentication
6. Learner onboarding
7. Curriculum
8. Missions
9. Assessment and submissions
10. Evidence and evaluation
11. Mastery Engine
12. Adaptive Roadmap
13. Rewards and streaks
14. AI Mentor
15. Secure sandbox
16. Collaboration
17. Projects
18. Complete frontend experience
19. Security and production hardening
20. Staging and production deployment

This README describes the approved project direction and must not be treated as permission to redesign the architecture.
