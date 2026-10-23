import { describe, expect, it } from "vitest";
import { onlyChanged } from "./subscriptionUtils";

describe("onlyChanged", () => {
  it("matches a schedule attachment without a plan change", () => {
    expect(onlyChanged({ schedule: null }, "schedule")).toBe(true);
  });

  it("does not hide a schedule update that also changes items", () => {
    expect(onlyChanged({ schedule: null, items: { data: [] } }, "schedule")).toBe(false);
  });

  it("does not match missing previous attributes", () => {
    expect(onlyChanged(undefined, "schedule")).toBe(false);
  });
});
