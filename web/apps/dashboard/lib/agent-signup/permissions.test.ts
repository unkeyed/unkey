import { describe, expect, it } from "vitest";
import { AgentSignupError } from "./errors";
import {
  AGENT_PERMISSION_ALLOWLIST,
  DEFAULT_AGENT_PERMISSIONS,
  resolveAgentPermissions,
} from "./permissions";

const workspaceId = "ws_test";

describe("resolveAgentPermissions", () => {
  it("uses a narrow default inside the allowlist", () => {
    for (const grant of DEFAULT_AGENT_PERMISSIONS) {
      expect(AGENT_PERMISSION_ALLOWLIST).toContainEqual(grant);
    }
    expect(resolveAgentPermissions(workspaceId, undefined)).toEqual([
      `unkey:v1:${workspaceId}:projects/*/keyspaces/*#read`,
      `unkey:v1:${workspaceId}:projects/*/keyspaces/*#write`,
      `unkey:v1:${workspaceId}:projects/*/keyspaces/*/keys/*#read`,
      `unkey:v1:${workspaceId}:projects/*/keyspaces/*/keys/*#write`,
      `unkey:v1:${workspaceId}:projects/*/keyspaces/*/keys/*#verify`,
    ]);
  });

  it("accepts an allowlisted subset and drops duplicates", () => {
    expect(
      resolveAgentPermissions(workspaceId, [
        { path: "projects/*/keyspaces/*/keys/*", action: "verify" },
        { path: "projects/*/keyspaces/*/keys/*", action: "verify" },
      ]),
    ).toEqual([`unkey:v1:${workspaceId}:projects/*/keyspaces/*/keys/*#verify`]);
  });

  it("rejects decrypt, root keys, and any other workspace", () => {
    expect(() =>
      resolveAgentPermissions(workspaceId, [
        { path: "projects/*/keyspaces/*/keys/*", action: "decrypt" },
      ]),
    ).toThrow(expect.objectContaining({ status: 400, code: "permission_denied" }));
    expect(() =>
      resolveAgentPermissions(workspaceId, [{ path: "rootKeys/*", action: "write" }]),
    ).toThrow(AgentSignupError);
    expect(() => resolveAgentPermissions(workspaceId, [{ path: "**", action: "write" }])).toThrow(
      AgentSignupError,
    );
  });
});
