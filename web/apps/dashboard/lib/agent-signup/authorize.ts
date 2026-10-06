import { AgentSignupError } from "./errors";

export type WorkspaceMembership = {
  organizationId: string;
  role: string;
  status: string;
};

export function assertWorkspaceAdmin(
  memberships: readonly WorkspaceMembership[],
  orgId: string,
): void {
  const membership = memberships.find((item) => item.organizationId === orgId);
  if (membership?.status === "active" && membership.role === "admin") {
    return;
  }
  throw new AgentSignupError(
    403,
    "forbidden",
    "You must be an admin of this workspace to create a root key.",
  );
}
