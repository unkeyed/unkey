import type Stripe from "stripe";
import { describe, expect, it } from "vitest";
import {
  cancellationFeedbackUpdate,
  isCancelJustScheduled,
  isCancellationFeedbackOnly,
  isScheduleUpdateOnly,
} from "./subscriptionUtils";

describe("isScheduleUpdateOnly", () => {
  it("matches a schedule attachment without a plan change", () => {
    expect(isScheduleUpdateOnly({ schedule: null })).toBe(true);
  });

  it("does not hide a schedule update that also changes items", () => {
    expect(isScheduleUpdateOnly({ schedule: null, items: { data: [] } })).toBe(false);
  });

  it("does not match missing previous attributes", () => {
    expect(isScheduleUpdateOnly(undefined)).toBe(false);
  });
});

describe("isCancelJustScheduled", () => {
  it("matches an update that set cancel_at_period_end", () => {
    expect(isCancelJustScheduled({ cancel_at_period_end: false })).toBe(true);
  });

  it("matches an update that set cancel_at", () => {
    expect(isCancelJustScheduled({ cancel_at: null })).toBe(true);
  });

  it("does not match an update that only records cancellation feedback", () => {
    expect(
      isCancelJustScheduled({
        cancellation_details: { comment: null, feedback: null, reason: null },
      }),
    ).toBe(false);
  });

  it("does not match missing previous attributes", () => {
    expect(isCancelJustScheduled(undefined)).toBe(false);
  });
});

describe("isCancellationFeedbackOnly", () => {
  const details = { comment: null, feedback: null, reason: null };

  it("matches an update that changed only the cancellation feedback", () => {
    expect(isCancellationFeedbackOnly({ cancellation_details: details })).toBe(true);
  });

  it("does not match a feedback change that also changed the cancel fields", () => {
    expect(
      isCancellationFeedbackOnly({ cancellation_details: details, cancel_at_period_end: false }),
    ).toBe(false);
  });

  it("does not match missing previous attributes", () => {
    expect(isCancellationFeedbackOnly(undefined)).toBe(false);
  });
});

describe("cancellationFeedbackUpdate", () => {
  function subWithDetails(details: Stripe.Subscription.CancellationDetails | null) {
    return { id: "sub_123", cancellation_details: details };
  }
  const previous = { cancellation_details: { comment: null, feedback: null, reason: null } };

  it("returns the reason and comment the update recorded", () => {
    const sub = subWithDetails({
      comment: "Too pricey for us",
      feedback: "too_expensive",
      reason: "cancellation_requested",
    });

    expect(cancellationFeedbackUpdate(previous, sub)).toEqual({
      reason: "too_expensive",
      comment: "Too pricey for us",
    });
  });

  it("returns a null comment when the customer left none", () => {
    const sub = subWithDetails({ comment: null, feedback: "unused", reason: null });

    expect(cancellationFeedbackUpdate(previous, sub)).toEqual({ reason: "unused", comment: null });
  });

  it("does not match an update that left cancellation_details alone", () => {
    const sub = subWithDetails({ comment: null, feedback: "unused", reason: null });

    expect(cancellationFeedbackUpdate({ cancel_at_period_end: false }, sub)).toBeNull();
    expect(cancellationFeedbackUpdate(undefined, sub)).toBeNull();
  });

  it("does not match when the feedback and comment did not change", () => {
    const sub = subWithDetails({
      comment: "Too pricey for us",
      feedback: "too_expensive",
      reason: "cancellation_requested",
    });
    const unchanged = {
      cancellation_details: {
        comment: "Too pricey for us",
        feedback: "too_expensive" as const,
        reason: null,
      },
    };

    expect(cancellationFeedbackUpdate(unchanged, sub)).toBeNull();
  });

  it("returns the reason and comment when only the comment changed", () => {
    const sub = subWithDetails({
      comment: "We moved to another vendor",
      feedback: "switched_service",
      reason: "cancellation_requested",
    });
    const commentOnly = {
      cancellation_details: {
        comment: "Too slow",
        feedback: "switched_service" as const,
        reason: "cancellation_requested" as const,
      },
    };

    expect(cancellationFeedbackUpdate(commentOnly, sub)).toEqual({
      reason: "switched_service",
      comment: "We moved to another vendor",
    });
  });

  it("does not match when the subscription carries no feedback", () => {
    const sub = subWithDetails({ comment: null, feedback: null, reason: "cancellation_requested" });

    expect(cancellationFeedbackUpdate(previous, sub)).toBeNull();
  });
});
