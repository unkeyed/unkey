import { describe, expect, it } from "vitest";
import { memoByArgs } from "./queries";

describe("memoByArgs", () => {
  it("returns the same value for the same arguments", () => {
    const memo = memoByArgs((projectId: string, appId: string | null) => ({ projectId, appId }));
    expect(memo("proj_KEBAP", "app_a")).toBe(memo("proj_KEBAP", "app_a"));
    expect(memo("proj_KEBAP", null)).not.toBe(memo("proj_KEBAP", "app_a"));
  });

  it("keeps only the most recently used values", () => {
    const memo = memoByArgs((id: string) => ({ id }));
    const first = memo("d_0");
    for (let i = 1; i <= 50; i++) {
      memo(`d_${i}`);
    }
    expect(memo("d_0")).not.toBe(first);

    const kept = memo("d_kept");
    for (let i = 0; i < 49; i++) {
      memo(`d_other_${i}`);
      memo("d_kept");
    }
    expect(memo("d_kept")).toBe(kept);
  });
});
