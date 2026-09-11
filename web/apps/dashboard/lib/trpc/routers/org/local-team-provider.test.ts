import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  authProvider: "local" as "workos" | "local",
}));

vi.mock("@/lib/env", () => ({
  env: () => ({ AUTH_PROVIDER: mocks.authProvider }),
}));

import { localAuth } from "@/lib/auth/local";
import { getLocalTeamProvider } from "./local-team-provider";

describe("getLocalTeamProvider", () => {
  beforeEach(() => {
    mocks.authProvider = "local";
  });

  it("returns the deterministic provider in local mode", () => {
    expect(getLocalTeamProvider()).toBe(localAuth);
  });

  it("does not expose custom team procedures in WorkOS mode", () => {
    mocks.authProvider = "workos";

    expect(() => getLocalTeamProvider()).toThrowError(
      expect.objectContaining({
        code: "NOT_FOUND",
        message: "Team management is handled by WorkOS",
      }),
    );
  });
});
