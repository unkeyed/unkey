import { isDuplicateKeyError } from "@unkey/db";
import { AgentSignupError } from "./errors";

export type SignupRow = {
  status: string;
  workspaceId: string | null;
  rootKeyId: string | null;
  workosUserId: string;
};

export type RootKeyDecision = "ok" | "missing" | "forbidden" | "conflict" | "pending";

export function rejectionForSignupInsert(error: unknown): AgentSignupError | null {
  if (!isDuplicateKeyError(error)) {
    return null;
  }
  return new AgentSignupError(
    409,
    "workspace_exists",
    "This agent registration already created a workspace.",
  );
}

export async function createClaimedWorkspace<T>(
  insert: () => Promise<void>,
  create: () => Promise<T>,
): Promise<T> {
  try {
    await insert();
  } catch (error) {
    const rejection = rejectionForSignupInsert(error);
    if (rejection) {
      throw rejection;
    }
    throw error;
  }
  return create();
}

export function decideRootKey(row: SignupRow | undefined, userId: string): RootKeyDecision {
  if (!row) {
    return "missing";
  }
  if (row.workosUserId !== userId) {
    return "forbidden";
  }
  if (row.status === "pending") {
    return "pending";
  }
  if (row.status === "workspace_created" && row.workspaceId && !row.rootKeyId) {
    return "ok";
  }
  return "conflict";
}

export function rootKeyDecisionError(decision: Exclude<RootKeyDecision, "ok">): AgentSignupError {
  switch (decision) {
    case "missing":
      return new AgentSignupError(
        404,
        "workspace_required",
        "Create a workspace with this agent registration before creating a root key.",
      );
    case "forbidden":
      return new AgentSignupError(
        403,
        "user_mismatch",
        "This agent registration is bound to a different user.",
      );
    case "pending":
      return new AgentSignupError(
        409,
        "workspace_pending",
        "Workspace creation for this agent registration has not finished.",
      );
    case "conflict":
      return new AgentSignupError(
        409,
        "root_key_exists",
        "This agent registration already received its first root key.",
      );
  }
}
