import type Stripe from "stripe";
import { describe, expect, it } from "vitest";
import {
  cancellationFeedbackUpdate,
  isCancelJustScheduled,
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

  it("does not match when the subscription carries no feedback", () => {
    const sub = subWithDetails({ comment: null, feedback: null, reason: "cancellation_requested" });

    expect(cancellationFeedbackUpdate(previous, sub)).toBeNull();
  });
});
