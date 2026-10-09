import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { beforeEach, describe, expect, it, vi } from "vitest";

type DeploySubscription = {
  plan: DeployPlan | null;
  signupCreditClaimed: boolean;
};

type Query = (opts: {
  ctx: {
    workspace: { id: string; deployPlan: string | null };
    user: { id: string };
  };
}) => Promise<DeploySubscription>;

const state = vi.hoisted(() => ({
  query: null as Query | null,
  existsForWorkspaceOrUser: vi.fn(),
  logOperation: vi.fn(),
}));

vi.mock("@/lib/logging", () => ({
  logOperation: state.logOperation,
}));

vi.mock("@/lib/stripe/computeSignupCreditClaims", () => ({
  drizzleComputeSignupCreditClaimStore: {
    existsForWorkspaceOrUser: state.existsForWorkspaceOrUser,
  },
}));

vi.mock("../../trpc", () => ({
  workspaceProcedure: {
    query: (resolver: Query) => {
      state.query = resolver;
      return resolver;
    },
  },
}));

import { getDeploySubscription } from "./getDeploySubscription";

function run(
  deployPlan: string | null,
  workspaceId = "ws_1",
  userId = "user_1",
): Promise<DeploySubscription> {
  const query = state.query;
  if (query === null) {
    throw new Error("getDeploySubscription did not register a query");
  }
  return query({
    ctx: { workspace: { id: workspaceId, deployPlan }, user: { id: userId } },
  });
}

describe("getDeploySubscription", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns signupCreditClaimed true when this workspace already has a claim", async () => {
    state.existsForWorkspaceOrUser.mockResolvedValue(true);

    await expect(run("pro", "ws_claimed", "user_other")).resolves.toEqual({
      plan: "pro",
      signupCreditClaimed: true,
    });
    expect(state.existsForWorkspaceOrUser).toHaveBeenCalledWith("ws_claimed", "user_other");
    expect(state.logOperation).not.toHaveBeenCalled();
    expect(getDeploySubscription).toBe(state.query);
  });

  it("returns signupCreditClaimed true when this user claimed on another workspace", async () => {
    state.existsForWorkspaceOrUser.mockResolvedValue(true);

    await expect(run("pro", "ws_new", "user_claimed")).resolves.toEqual({
      plan: "pro",
      signupCreditClaimed: true,
    });
    expect(state.existsForWorkspaceOrUser).toHaveBeenCalledWith("ws_new", "user_claimed");
  });

  it("returns signupCreditClaimed false for a user and workspace with no claim", async () => {
    state.existsForWorkspaceOrUser.mockResolvedValue(false);

    await expect(run("starter", "ws_new", "user_new")).resolves.toEqual({
      plan: "starter",
      signupCreditClaimed: false,
    });
    expect(state.existsForWorkspaceOrUser).toHaveBeenCalledWith("ws_new", "user_new");
    expect(state.logOperation).not.toHaveBeenCalled();
  });

  it("returns the plan with signupCreditClaimed false when the claim read throws", async () => {
    state.existsForWorkspaceOrUser.mockRejectedValue(new Error("primary unavailable"));

    await expect(run("business")).resolves.toEqual({
      plan: "business",
      signupCreditClaimed: false,
    });
    expect(state.logOperation).toHaveBeenCalledTimes(1);
    expect(state.logOperation).toHaveBeenCalledWith(
      "error",
      "Failed to read compute signup credit claim",
      {
        workspace_id: "ws_1",
        error_message: "primary unavailable",
      },
    );
  });
});
