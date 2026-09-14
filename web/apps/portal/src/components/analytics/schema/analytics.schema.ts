import { z } from "zod";

/** The API buckets hourly up to this many days and daily beyond. */
export const HOURLY_MAX_DAYS = 4;

/** Rejection outcomes `v2/portal.getVerifications` breaks out per bucket. */
export const OUTCOME_KINDS = [
  "rateLimited",
  "expired",
  "disabled",
  "usageExceeded",
  "insufficientPermissions",
  "forbidden",
] as const;
export type OutcomeKind = (typeof OUTCOME_KINDS)[number];
export type OutcomeCounts = Record<OutcomeKind, number>;

/**
 * Ceiling on any verifications window. The API caps the window at the
 * workspace's retention, but treats a non-positive retention as "no check", so
 * the portal never asks for an unbounded range.
 */
export const MAX_WINDOW_DAYS = 365;

/** Query params for `v2/portal.getVerifications`: a window in unix ms, optionally narrowed to one key. */
export const getVerificationsQuerySchema = z
  .object({
    startTime: z.number().int().nonnegative(),
    endTime: z.number().int().positive(),
    keyId: z.string().min(1).optional(),
  })
  .refine((q) => q.endTime > q.startTime, { message: "endTime must be after startTime" })
  .refine((q) => q.endTime - q.startTime <= MAX_WINDOW_DAYS * 86_400_000, {
    message: `time window must not exceed ${MAX_WINDOW_DAYS} days`,
  });

export type GetVerificationsQuery = z.infer<typeof getVerificationsQuerySchema>;

/**
 * One bucket of the verification timeseries, mapped from a
 * `v2/portal.getVerifications` data point. `error` is every rejection —
 * the six the API breaks out plus `other` — so the valid/invalid split, the
 * per-outcome breakdown and `total` always agree.
 */
export type VerificationBucket = OutcomeCounts & {
  /** Bucket start as a unix timestamp in milliseconds. */
  time: number;
  /** Total verifications in this bucket, across all outcomes. */
  total: number;
  /** Verifications with a VALID outcome. */
  valid: number;
  /** Non-valid verifications (rate limited, forbidden, expired, etc.). */
  error: number;
  /** Rejections counted in `total` that the API does not break out. */
  other: number;
};
