/**
 * Shared TypeScript types that mirror the Go API response shapes.
 *
 * These are read-only value types — no business logic lives here.
 * The frontend never directly mutates authoritative learner state.
 */

// ─── Auth / Identity ─────────────────────────────────────────────────────────

export interface User {
  id: string;
  email: string;
  role: string;
}

// ─── Learner ──────────────────────────────────────────────────────────────────

export interface LearnerProfile {
  user_id: string;
  display_name: string;
}

// ─── API Errors ──────────────────────────────────────────────────────────────

/** Structured error returned by every non-success API response. */
export interface ApiError {
  error: string;
}

/** Well-known error codes returned by the API. */
export type ApiErrorCode =
  | "invalid_request"
  | "unauthorized"
  | "rate_limited"
  | "email_already_registered"
  | "learner_profile_exists"
  | "not_found"
  | "internal_error"
  | string; // allow unknown codes to be handled gracefully
