# ZERO → AI ENGINEER Web

The Next.js application provides registration, sign-in, learner onboarding, and
an authenticated view of the published curriculum through the Go API.

## Local development

Run the API on `http://localhost:8080`, then start the web application from this directory:

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). The development API proxy forwards `/api/*` requests to `http://localhost:8080`.

## Vercel deployment

Configure this directory (`apps/web`) as the Vercel project root. Set `API_ORIGIN` in the Vercel project environment to the public HTTPS origin of the Go API. Production builds fail if this setting is missing or is not HTTPS.

The app proxies `/api/*` through the Vercel origin. This keeps the API session cookie same-origin in the browser, while the Go API must set `WEB_ORIGIN` to the exact Vercel origin and allow that origin through its CORS/Origin policy.

After sign-in and learner onboarding, the app loads the current curriculum from
`GET /api/v1/curriculum` and displays its milestones, skills, objectives, and
prerequisites. The API also provides the authenticated, read-only
`GET /api/v1/curriculum/{version}` endpoint for a specific published version.

Set the Go API's database URL and session secret in the API host's secret manager. Do not set database credentials or session secrets in Vercel's `NEXT_PUBLIC_*` variables.

Validate before deploying:

```bash
npm run lint
npx tsc --noEmit
npm run build
```
