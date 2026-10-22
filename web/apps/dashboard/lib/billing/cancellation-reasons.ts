import type Stripe from "stripe";

export const CANCELLATION_COMMENT_MAX_LENGTH = 2000;

export type CancellationReason = Stripe.SubscriptionUpdateParams.CancellationDetails.Feedback;

export const CANCELLATION_REASON_LABELS: Record<CancellationReason, string> = {
  too_expensive: "Too expensive",
  missing_features: "Missing a feature I need",
  switched_service: "Moved to another tool",
  unused: "Not using it enough",
  too_complex: "Too hard to use",
  low_quality: "Quality or reliability issues",
  customer_service: "Support experience",
  other: "Something else",
};

export function isCancellationReason(value: string): value is CancellationReason {
  return Object.hasOwn(CANCELLATION_REASON_LABELS, value);
}

export const CANCELLATION_REASONS: CancellationReason[] = Object.keys(
  CANCELLATION_REASON_LABELS,
).filter(isCancellationReason);
