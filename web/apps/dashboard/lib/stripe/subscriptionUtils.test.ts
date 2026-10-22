import { describe, expect, it } from "vitest";
import { isCancelJustScheduled, isScheduleUpdateOnly } from "./subscriptionUtils";

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
