"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { FormEvent } from "react";

type PublicUser = {
  id: string;
  email: string;
  role: string;
};

type LearnerProfile = {
  user_id: string;
  display_name: string;
};

type CurriculumSkill = {
  id: string;
  title: string;
  objectives: string[];
  prerequisites: string[];
};

type Curriculum = {
  schema_version: number;
  version: string;
  title: string;
  skills: CurriculumSkill[];
  milestones: {
    id: string;
    title: string;
    skill_ids: string[];
  }[];
  activities: {
    id: string;
    kind: string;
    skill_ids: string[];
  }[];
};

type Screen =
  | { kind: "loading" }
  | { kind: "auth"; notice?: string }
  | { kind: "onboarding"; user: PublicUser }
  | { kind: "ready"; user: PublicUser; learner: LearnerProfile; curriculum: Curriculum }
  | { kind: "error"; message: string };

class APIError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
  ) {
    super(code);
  }
}

async function requestJSON(path: string, init: RequestInit = {}): Promise<unknown> {
  const response = await fetch(path, {
    ...init,
    credentials: "include",
    headers: {
      ...(init.body ? { "Content-Type": "application/json" } : {}),
      ...init.headers,
    },
  });

  if (!response.ok) {
    let code = "internal_error";
    try {
      const payload: unknown = await response.json();
      if (
        typeof payload === "object" &&
        payload !== null &&
        "error" in payload &&
        typeof payload.error === "string"
      ) {
        code = payload.error;
      }
    } catch {
      throw new Error("The API returned an invalid error response.");
    }
    throw new APIError(response.status, code);
  }

  return response.json();
}

function readPublicUser(payload: unknown): PublicUser {
  if (
    typeof payload !== "object" ||
    payload === null ||
    !("user" in payload) ||
    typeof payload.user !== "object" ||
    payload.user === null ||
    !("id" in payload.user) ||
    typeof payload.user.id !== "string" ||
    !("email" in payload.user) ||
    typeof payload.user.email !== "string" ||
    !("role" in payload.user) ||
    typeof payload.user.role !== "string"
  ) {
    throw new Error("The API returned an invalid user response.");
  }

  return {
    id: payload.user.id,
    email: payload.user.email,
    role: payload.user.role,
  };
}

function readLearner(payload: unknown): LearnerProfile {
  if (
    typeof payload !== "object" ||
    payload === null ||
    !("learner" in payload) ||
    typeof payload.learner !== "object" ||
    payload.learner === null ||
    !("user_id" in payload.learner) ||
    typeof payload.learner.user_id !== "string" ||
    !("display_name" in payload.learner) ||
    typeof payload.learner.display_name !== "string"
  ) {
    throw new Error("The API returned an invalid learner response.");
  }

  return {
    user_id: payload.learner.user_id,
    display_name: payload.learner.display_name,
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item: unknown) => typeof item === "string");
}

function readSkill(payload: unknown): CurriculumSkill {
  if (
    !isRecord(payload) ||
    typeof payload.id !== "string" ||
    typeof payload.title !== "string" ||
    !isStringArray(payload.objectives) ||
    !isStringArray(payload.prerequisites)
  ) {
    throw new Error("The API returned an invalid curriculum response.");
  }
  return {
    id: payload.id,
    title: payload.title,
    objectives: payload.objectives,
    prerequisites: payload.prerequisites,
  };
}

function readMilestone(payload: unknown): Curriculum["milestones"][number] {
  if (
    !isRecord(payload) ||
    typeof payload.id !== "string" ||
    typeof payload.title !== "string" ||
    !isStringArray(payload.skill_ids)
  ) {
    throw new Error("The API returned an invalid curriculum response.");
  }
  return { id: payload.id, title: payload.title, skill_ids: payload.skill_ids };
}

function readActivity(payload: unknown): Curriculum["activities"][number] {
  if (
    !isRecord(payload) ||
    typeof payload.id !== "string" ||
    typeof payload.kind !== "string" ||
    !isStringArray(payload.skill_ids)
  ) {
    throw new Error("The API returned an invalid curriculum response.");
  }
  return { id: payload.id, kind: payload.kind, skill_ids: payload.skill_ids };
}

function readArray<T>(payload: unknown, readItem: (item: unknown) => T): T[] {
  if (!Array.isArray(payload)) {
    throw new Error("The API returned an invalid curriculum response.");
  }
  return payload.map(readItem);
}

function readCurriculum(payload: unknown): Curriculum {
  if (
    !isRecord(payload) ||
    payload.schema_version !== 1 ||
    typeof payload.version !== "string" ||
    typeof payload.title !== "string"
  ) {
    throw new Error("The API returned an invalid curriculum response.");
  }

  const curriculum: Curriculum = {
    schema_version: payload.schema_version,
    version: payload.version,
    title: payload.title,
    skills: readArray(payload.skills, readSkill),
    milestones: readArray(payload.milestones, readMilestone),
    activities: readArray(payload.activities, readActivity),
  };
  const skillIDs = new Set(curriculum.skills.map((skill) => skill.id));
  if (
    curriculum.milestones.some((milestone) => milestone.skill_ids.some((skillID) => !skillIDs.has(skillID))) ||
    curriculum.skills.some((skill) => skill.prerequisites.some((skillID) => !skillIDs.has(skillID))) ||
    curriculum.activities.some((activity) => activity.skill_ids.some((skillID) => !skillIDs.has(skillID)))
  ) {
    throw new Error("The API returned an invalid curriculum response.");
  }
  return curriculum;
}

async function loadAuthenticatedScreen(): Promise<Screen> {
  const user = readPublicUser(await requestJSON("/api/v1/me"));

  try {
    const learner = readLearner(await requestJSON("/api/v1/learners/me"));
    const curriculum = readCurriculum(await requestJSON("/api/v1/curriculum"));
    return { kind: "ready", user, learner, curriculum };
  } catch (error) {
    if (error instanceof APIError && error.status === 404) {
      return { kind: "onboarding", user };
    }
    throw error;
  }
}

function errorMessage(error: unknown, authMode?: "login" | "register"): string {
  if (error instanceof APIError) {
    switch (error.code) {
      case "invalid_request":
        if (authMode === "register") {
          return "Enter a valid email address and a password between 12 and 128 characters.";
        }
        return "Check your details and try again.";
      case "email_already_registered":
        return "An account with this email already exists. Sign in instead.";
      case "unauthorized":
        return "Those sign-in details were not recognised.";
      case "rate_limited":
        return "There have been too many attempts. Wait a moment and try again.";
      case "learner_profile_exists":
        return "Your learner profile already exists. Refresh to continue.";
      default:
        return "The request could not be completed. Try again shortly.";
    }
  }
  return "The service could not be reached. Check your connection and try again.";
}

export default function Home() {
  const [screen, setScreen] = useState<Screen>({ kind: "loading" });
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let active = true;

    void loadAuthenticatedScreen()
      .then((nextScreen) => {
        if (active) setScreen(nextScreen);
      })
      .catch((error: unknown) => {
        if (!active) return;
        if (error instanceof APIError && error.status === 401) {
          setScreen({ kind: "auth" });
          return;
        }
        setScreen({
          kind: "error",
          message: "We could not load your account. Check that the API is running, then try again.",
        });
      });

    return () => {
      active = false;
    };
  }, []);

  async function submitAuthentication(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (mode === "register") {
      const passwordLength = Array.from(password).length;
      if (passwordLength < 12 || passwordLength > 128) {
        setMessage("Your password must be between 12 and 128 characters.");
        return;
      }
    }

    setBusy(true);
    setMessage("");

    try {
      if (mode === "register") {
        await requestJSON("/api/v1/auth/register", {
          method: "POST",
          body: JSON.stringify({ email, password }),
        });
        setPassword("");
        setMode("login");
        setMessage("Account created. Sign in to continue.");
        setScreen({ kind: "auth" });
        return;
      }

      await requestJSON("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      setPassword("");
      setScreen(await loadAuthenticatedScreen());
    } catch (error) {
      setMessage(errorMessage(error, mode));
    } finally {
      setBusy(false);
    }
  }

  async function submitOnboarding(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (screen.kind !== "onboarding") return;

    setBusy(true);
    setMessage("");
    try {
      await requestJSON("/api/v1/learners/onboard", {
        method: "POST",
        body: JSON.stringify({ display_name: displayName }),
      });
      setScreen(await loadAuthenticatedScreen());
    } catch (error) {
      setMessage(errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function signOut() {
    setBusy(true);
    setMessage("");
    try {
      const response = await fetch("/api/v1/auth/logout", {
        method: "POST",
        credentials: "include",
      });
      if (!response.ok) {
        throw new APIError(response.status, "internal_error");
      }
      setMode("login");
      setPassword("");
      setScreen({ kind: "auth", notice: "You have signed out." });
    } catch (error) {
      setMessage(errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <Link className="wordmark" href="/" aria-label="Zero to AI Engineer home">
          <span className="wordmark-mark" aria-hidden="true">Z</span>
          <span>ZERO <span className="wordmark-arrow">→</span> AI ENGINEER</span>
        </Link>
        <span className="topbar-label">YOUR AI ENGINEERING APPRENTICESHIP</span>
      </header>

      <section className="hero-grid">
        <div className="hero-copy">
          <p className="eyebrow"><span className="eyebrow-dot" /> LEARN BY BUILDING</p>
          <h1>Become the engineer<br />who can <span>build what&apos;s next.</span></h1>
          <p className="hero-description">
            A focused path from your first lines of code to building real AI systems.
            Learn by practising, building, and showing what you can do.
          </p>
          <div className="journey-list" aria-label="Learning approach">
            <div className="journey-item"><span className="journey-number">01</span><span><strong>Learn the foundations</strong><small>Build understanding one step at a time</small></span></div>
            <div className="journey-item"><span className="journey-number">02</span><span><strong>Practise with purpose</strong><small>Turn concepts into working skills</small></span></div>
            <div className="journey-item"><span className="journey-number">03</span><span><strong>Build real things</strong><small>Make your progress visible through evidence</small></span></div>
          </div>
          <p className="hero-footnote">YOUR STARTING POINT DOESN&apos;T DEFINE YOUR DESTINATION.</p>
        </div>

        <div className="account-panel">
          {screen.kind === "loading" && (
            <div className="panel-loading" role="status">Checking your session…</div>
          )}

          {screen.kind === "error" && (
            <div className="panel-content">
              <p className="eyebrow">CONNECTION ISSUE</p>
              <h2>We couldn&apos;t reach your account.</h2>
              <p className="panel-description">{screen.message}</p>
              <button className="button button-primary" onClick={() => window.location.reload()}>
                Try again
              </button>
            </div>
          )}

          {screen.kind === "auth" && (
            <div className="panel-content">
              <p className="eyebrow">{mode === "login" ? "WELCOME BACK" : "START YOUR JOURNEY"}</p>
              <h2>{mode === "login" ? "Sign in to continue." : "Create your account."}</h2>
              <p className="panel-description">
                {mode === "login"
                  ? "Pick up where your learning journey left off."
                  : "One account to keep your learning progress together."}
              </p>
              {screen.notice && <p className="notice" role="status">{screen.notice}</p>}
              {message && <p className="form-message" role="alert">{message}</p>}
              <form className="account-form" onSubmit={submitAuthentication}>
                <label htmlFor="email">Email address</label>
                <input
                  autoComplete="email"
                  id="email"
                  name="email"
                  onChange={(event) => setEmail(event.target.value)}
                  required
                  type="email"
                  value={email}
                />
                <label htmlFor="password">Password</label>
                <input
                  autoComplete={mode === "login" ? "current-password" : "new-password"}
                  id="password"
                  name="password"
                  onChange={(event) => setPassword(event.target.value)}
                  required
                  aria-describedby={mode === "register" ? "password-requirements" : undefined}
                  type="password"
                  value={password}
                />
                {mode === "register" && (
                  <p className="field-hint" id="password-requirements">
                    Use 12–128 characters. Passwords are case-sensitive.
                  </p>
                )}
                <button className="button button-primary" disabled={busy} type="submit">
                  {busy ? "Please wait…" : mode === "login" ? "Sign in" : "Create account"}
                  {!busy && <span aria-hidden="true">↗</span>}
                </button>
              </form>
              <p className="mode-switch">
                {mode === "login" ? "New to the journey?" : "Already have an account?"}{" "}
                <button
                  disabled={busy}
                  onClick={() => {
                    setMode(mode === "login" ? "register" : "login");
                    setMessage("");
                  }}
                  type="button"
                >
                  {mode === "login" ? "Create an account" : "Sign in"}
                </button>
              </p>
              <p className="privacy-note">Your password is sent to the API for verification; this page does not persist it.</p>
            </div>
          )}

          {screen.kind === "onboarding" && (
            <div className="panel-content">
              <p className="eyebrow">FIRST, A LITTLE ABOUT YOU</p>
              <h2>What should we call you?</h2>
              <p className="panel-description">Choose a display name for your learner profile.</p>
              {message && <p className="form-message" role="alert">{message}</p>}
              <form className="account-form" onSubmit={submitOnboarding}>
                <label htmlFor="display-name">Display name</label>
                <input
                  autoComplete="nickname"
                  id="display-name"
                  onChange={(event) => setDisplayName(event.target.value)}
                  required
                  type="text"
                  value={displayName}
                />
                <button className="button button-primary" disabled={busy} type="submit">
                  {busy ? "Saving…" : "Continue"}
                  {!busy && <span aria-hidden="true">↗</span>}
                </button>
              </form>
              <button className="text-button signout-button" disabled={busy} onClick={signOut} type="button">
                Sign out
              </button>
            </div>
          )}

          {screen.kind === "ready" && (
            <div className="panel-content profile-content">
              <p className="eyebrow"><span className="eyebrow-dot" /> PROFILE READY</p>
              <h2>Good to have you here, {screen.learner.display_name}.</h2>
              <p className="panel-description">
                Your learner profile is set up. Explore the first skills in your learning path.
              </p>
              <div className="profile-card">
                <span className="profile-avatar" aria-hidden="true">
                  {screen.learner.display_name.slice(0, 1).toUpperCase()}
                </span>
                <span className="profile-details">
                  <strong>{screen.learner.display_name}</strong>
                  <small>{screen.user.email}</small>
                </span>
                <span className="profile-status">ACTIVE</span>
              </div>
              <section className="curriculum-section" aria-labelledby="curriculum-title">
                <div className="curriculum-heading">
                  <div>
                    <p className="eyebrow">YOUR CURRICULUM</p>
                    <h3 id="curriculum-title">{screen.curriculum.title}</h3>
                  </div>
                  <span className="curriculum-version">v{screen.curriculum.version}</span>
                </div>
                {screen.curriculum.milestones.map((milestone) => (
                  <section className="curriculum-milestone" key={milestone.id}>
                    <h4>{milestone.title}</h4>
                    <ol className="curriculum-skills">
                      {milestone.skill_ids.map((skillID, index) => {
                        const skill = screen.curriculum.skills.find((item) => item.id === skillID);
                        if (!skill) return null;
                        return (
                          <li className="curriculum-skill" key={skill.id}>
                            <span className="curriculum-skill-number">{String(index + 1).padStart(2, "0")}</span>
                            <div>
                              <h5>{skill.title}</h5>
                              <ul className="curriculum-objectives">
                                {skill.objectives.map((objective) => <li key={objective}>{objective}</li>)}
                              </ul>
                              {skill.prerequisites.length > 0 && (
                                <small>
                                  Builds on: {skill.prerequisites
                                    .map((id) => screen.curriculum.skills.find((item) => item.id === id)?.title ?? id)
                                    .join(", ")}
                                </small>
                              )}
                            </div>
                          </li>
                        );
                      })}
                    </ol>
                  </section>
                ))}
              </section>
              {message && <p className="form-message" role="alert">{message}</p>}
              <button className="button button-secondary" disabled={busy} onClick={signOut} type="button">
                {busy ? "Signing out…" : "Sign out"}
              </button>
              <p className="privacy-note">Your profile and progress are managed securely by the platform.</p>
            </div>
          )}
        </div>
      </section>

      <footer className="page-footer">
        <span>ZERO → AI ENGINEER</span>
        <span>COMPETENCE OVER COURSE COMPLETION.</span>
      </footer>
    </main>
  );
}
