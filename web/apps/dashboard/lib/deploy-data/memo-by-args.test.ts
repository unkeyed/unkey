import { describe, expect, it } from "vitest";
import { memoByArgs } from "./queries";

describe("memoByArgs", () => {
  it("returns the same value for the same arguments", () => {
    const memo = memoByArgs((projectId: string, appId: string | null) => ({ projectId, appId }));
    expect(memo("proj_KEBAP", "app_a")).toBe(memo("proj_KEBAP", "app_a"));
    expect(memo("proj_KEBAP", null)).not.toBe(memo("proj_KEBAP", "app_a"));
  });
});
