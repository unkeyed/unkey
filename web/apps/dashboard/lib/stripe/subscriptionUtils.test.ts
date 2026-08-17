import { describe, expect, it } from "vitest";
import { isScheduleUpdateOnly } from "./subscriptionUtils";

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
