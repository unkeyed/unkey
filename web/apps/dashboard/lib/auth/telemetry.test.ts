import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  logOperation: vi.fn(),
}));

vi.mock("@/lib/logging", () => ({
  logOperation: mocks.logOperation,
}));

import {
  SLOW_SESSION_VALIDATION_MS,
  logManagedAuthOutcome,
  logSessionValidationDuration,
} from "./telemetry";

describe("logManagedAuthOutcome", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("records a success at info level", () => {
    logManagedAuthOutcome("session_refresh", "success");

    expect(mocks.logOperation).toHaveBeenCalledWith("info", "Managed authentication outcome", {
      auth_event: "session_refresh",
      auth_outcome: "success",
    });
  });

  it.each(["failure", "slow"] as const)("records a %s at warn level", (outcome) => {
    logManagedAuthOutcome("session_refresh", outcome);

    expect(mocks.logOperation).toHaveBeenCalledWith("warn", "Managed authentication outcome", {
      auth_event: "session_refresh",
      auth_outcome: outcome,
    });
  });
});

describe("logSessionValidationDuration", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("stays silent for a validation inside the bound", () => {
    logSessionValidationDuration(SLOW_SESSION_VALIDATION_MS - 1);

    expect(mocks.logOperation).not.toHaveBeenCalled();
  });

  it("reports a stalled validation as a warn-level session refresh outcome", () => {
    logSessionValidationDuration(SLOW_SESSION_VALIDATION_MS);

    expect(mocks.logOperation).toHaveBeenCalledWith("warn", "Managed authentication outcome", {
      auth_event: "session_refresh",
      auth_outcome: "slow",
    });
  });
});
