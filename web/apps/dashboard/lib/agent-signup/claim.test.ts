import { describe, expect, it, vi } from "vitest";
import { createClaimedWorkspace, decideRootKey, rootKeyDecisionError } from "./claim";
import { AgentSignupError } from "./errors";

describe("one workspace per claim", () => {
  it("does not create a second workspace when the registration insert conflicts", async () => {
    const taken = new Set<string>();
    const create = vi.fn(async () => "created");
    const insert = async () => {
      if (taken.has("agent_reg_1")) {
        const error = new Error("duplicate key");
        Object.assign(error, { errno: 1062, code: "ER_DUP_ENTRY" });
        throw error;
      }
      taken.add("agent_reg_1");
    };

    await expect(createClaimedWorkspace(insert, create)).resolves.toBe("created");
    await expect(createClaimedWorkspace(insert, create)).rejects.toMatchObject({
      status: 409,
      code: "workspace_exists",
    });
    expect(create).toHaveBeenCalledTimes(1);
  });
});

describe("decideRootKey", () => {
  const row = {
    status: "workspace_created",
    workspaceId: "ws_1",
    rootKeyId: null,
    workosUserId: "user_1",
  };

  it("allows the first key for the claiming user only", () => {
    expect(decideRootKey(row, "user_1")).toBe("ok");
    expect(decideRootKey(undefined, "user_1")).toBe("missing");
    expect(decideRootKey(row, "user_2")).toBe("forbidden");
    expect(decideRootKey({ ...row, status: "pending" }, "user_1")).toBe("pending");
    expect(decideRootKey({ ...row, status: "root_key_issued", rootKeyId: "key_1" }, "user_1")).toBe(
      "conflict",
    );
  });

  it("maps a second issuance to a conflict the caller can return", () => {
    const error = rootKeyDecisionError("conflict");
    expect(error).toBeInstanceOf(AgentSignupError);
    expect(error.status).toBe(409);
    expect(error.code).toBe("root_key_exists");
  });
});
