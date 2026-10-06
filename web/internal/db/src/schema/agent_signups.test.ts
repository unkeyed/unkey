import { describe, expect, it } from "vitest";
import { agentSignups } from "./agent_signups";

describe("agent_signups", () => {
  it("allows one row per agent registration", () => {
    expect(agentSignups.agentRegistrationId.isUnique).toBe(true);
    expect(agentSignups.id.isUnique).toBe(true);
    expect(agentSignups.rootKeyId.notNull).toBe(false);
    expect(agentSignups.workspaceId.notNull).toBe(false);
  });
});
