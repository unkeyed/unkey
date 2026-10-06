import { mintProxyJWT } from "@/lib/auth/proxy-jwt";
import { auth } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { createFreeWorkspaceInTx } from "@/lib/workspace/create-workspace";
import { and, eq, schema } from "@unkey/db";
import { z } from "zod";
import { assertWorkspaceAdmin } from "./authorize";
import { AgentSignupError } from "./errors";
import { type PermissionGrant, resolveAgentPermissions } from "./permissions";
import type { VerifiedAgent } from "./verify";

const rootKeyResponse = z.object({
  data: z.object({
    keyId: z.string().min(1),
    key: z.string().min(1),
  }),
});

const AGENT_REGISTRATION_METADATA_KEY = "agent_registration_id";

export async function createAgentWorkspace(input: {
  agent: VerifiedAgent;
  name: string;
  slug: string;
  audit: { location: string; userAgent?: string };
}) {
  return db.transaction((tx) =>
    createFreeWorkspaceInTx(tx, {
      name: input.name,
      slug: input.slug,
      userId: input.agent.userId,
      audit: input.audit,
      metadata: {
        [AGENT_REGISTRATION_METADATA_KEY]: input.agent.registrationId,
      },
      createTenant: (params) => auth.createTenant(params),
      localOrgId: null,
    }),
  );
}

async function createRootKey(input: {
  orgId: string;
  userId: string;
  name: string;
  permissions: string[];
}): Promise<{ keyId: string; key: string }> {
  let token: string;
  try {
    token = await mintProxyJWT({
      orgId: input.orgId,
      role: "admin",
      subject: input.userId,
      name: input.userId,
    });
  } catch {
    throw new AgentSignupError(503, "not_configured", "Root key signing is not configured.");
  }

  let response: Response;
  try {
    response = await fetch(new URL("/v2/rootKeys.createKey", env().UNKEY_API_URL), {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
        "x-unkey-client": "unkey-dashboard",
      },
      body: JSON.stringify({
        name: input.name,
        permissions: input.permissions,
      }),
      signal: AbortSignal.timeout(10_000),
    });
  } catch {
    throw new AgentSignupError(503, "upstream", "The root key API could not be reached.");
  }
  if (!response.ok) {
    throw new AgentSignupError(502, "root_key_rejected", "Root key creation was rejected.");
  }
  const parsed = rootKeyResponse.safeParse(await response.json().catch(() => null));
  if (!parsed.success) {
    throw new AgentSignupError(
      502,
      "root_key_rejected",
      "Root key creation returned an unreadable response.",
    );
  }
  return parsed.data.data;
}

async function findAgentWorkspace(workspaceId?: string, slug?: string) {
  const filters = [
    ...(workspaceId ? [eq(schema.workspaces.id, workspaceId)] : []),
    ...(slug ? [eq(schema.workspaces.slug, slug)] : []),
  ];
  if (filters.length === 0) {
    throw new AgentSignupError(
      400,
      "invalid_body",
      "Send workspaceId or slug. name must be 1 to 256 characters. permissions must be a list of path and action.",
    );
  }
  const [workspace] = await db
    .select({
      id: schema.workspaces.id,
      orgId: schema.workspaces.orgId,
      slug: schema.workspaces.slug,
    })
    .from(schema.workspaces)
    .where(and(...filters))
    .limit(1);
  if (!workspace) {
    throw new AgentSignupError(
      404,
      "workspace_not_found",
      "No workspace matches that workspaceId and slug.",
    );
  }
  return workspace;
}

export async function issueAgentRootKey(input: {
  agent: VerifiedAgent;
  workspaceId?: string;
  slug?: string;
  name?: string;
  permissions?: readonly PermissionGrant[];
}) {
  const workspace = await findAgentWorkspace(input.workspaceId, input.slug);
  const memberships = await auth.listMemberships(input.agent.userId, workspace.orgId);
  assertWorkspaceAdmin(
    memberships.data.map((membership) => ({
      organizationId: membership.organization.id,
      role: membership.role,
      status: membership.status,
    })),
    workspace.orgId,
  );

  const permissions = resolveAgentPermissions(workspace.id, input.permissions);
  const created = await createRootKey({
    orgId: workspace.orgId,
    userId: input.agent.userId,
    name: input.name ?? "Agent",
    permissions,
  });
  return {
    keyId: created.keyId,
    key: created.key,
    permissions,
  };
}
