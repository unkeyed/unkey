import { describe, expect, test, vi } from "vitest";
import { safeResolve } from "./safe-resolve";

describe("safeResolve", () => {
  test("returns the resolved flag value", async () => {
    await expect(safeResolve("example", async () => true, false)).resolves.toBe(true);
  });

  test("returns the fallback when flag evaluation fails", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});

    await expect(
      safeResolve(
        "example",
        async () => {
          throw new Error("provider unavailable");
        },
        false,
      ),
    ).resolves.toBe(false);

    expect(warn).toHaveBeenCalledOnce();
    warn.mockRestore();
  });
});
