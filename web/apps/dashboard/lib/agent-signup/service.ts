import { auth } from "@/lib/auth/server";
import { primaryDb } from "@/lib/db";
import { env } from "@/lib/env";
import { createFreeWorkspace } from "@/lib/workspace/create-workspace";
import { and, eq, isNull, schema } from "@unkey/db";
import { z } from "zod";
import { assertWorkspaceAdmin } from "./authorize";
import { mintAgentSignupJWT } from "./credential";
import { AgentSignupError } from "./errors";
import { type PermissionGrant, resolveAgentPermissions } from "./permissions";
import type { VerifiedAgent } from "./verify";

const rootKeyResponse = z.object({
  data: z.object({
    keyId: z.string().min(1),
    key: z.string().min(1),
  }),
});

const upstreamErrorBody = z.object({
  meta: z
    .object({
      requestId: z.string().optional(),
    })
    .optional(),
  error: z
    .object({
      type: z.string().optional(),
      title: z.string().optional(),
      detail: z.string().optional(),
    })
    .optional(),
});

function upstreamErrorFields(body: unknown): {
  type?: string;
  title?: string;
  detail?: string;
  requestId?: string;
} {
  const parsed = upstreamErrorBody.safeParse(body);
  if (!parsed.success) {
    return {};
  }
  return {
    type: parsed.data.error?.type,
    title: parsed.data.error?.title,
    detail: parsed.data.error?.detail,
    requestId: parsed.data.meta?.requestId,
  };
}

const AGENT_REGISTRATION_METADATA_KEY = "agent_registration_id";

export async function createAgentWorkspace(input: {
  agent: VerifiedAgent;
  name: string;
  slug: string;
  audit: { location: string; userAgent?: string };
}) {
  return createFreeWorkspace({
    name: input.name,
    slug: input.slug,
    userId: input.agent.userId,
    audit: input.audit,
    metadata: {
      [AGENT_REGISTRATION_METADATA_KEY]: input.agent.registrationId,
    },
    createTenant: (params) => auth.createTenant(params),
    deleteTenant: (orgId) => auth.deleteTenant(orgId),
    localOrgId: null,
  });
}

async function createRootKey(input: {
  orgId: string;
  userId: string;
  name: string;
  permissions: string[];
}): Promise<{ keyId: string; key: string }> {
  let token: string;
  try {
    token = await mintAgentSignupJWT({
      orgId: input.orgId,
      subject: input.userId,
      name: input.userId,
    });
  } catch {
    throw new AgentSignupError(503, "not_configured", "Root key signing is not configured.");
  }

  let response: Response;
  try {
    response = await fetch(new URL("/v2/rootKeys.createAgentKey", env().UNKEY_API_URL), {
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
  } catch (error) {
    console.error("agent root key upstream unreachable", {
      reason: error instanceof Error ? error.name : "unknown",
    });
    throw new AgentSignupError(503, "upstream", "The root key API could not be reached.");
  }
  if (!response.ok) {
    const body: unknown = await response.json().catch(() => null);
    console.error("agent root key rejected", {
      status: response.status,
      ...upstreamErrorFields(body),
    });
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

export function liveWorkspaceWhere(workspaceId?: string, slug?: string) {
  const identity = [
    ...(workspaceId ? [eq(schema.workspaces.id, workspaceId)] : []),
    ...(slug ? [eq(schema.workspaces.slug, slug)] : []),
  ];
  if (identity.length === 0) {
    return undefined;
  }
  return and(
    isNull(schema.workspaces.deletedAtM),
    eq(schema.workspaces.enabled, true),
    ...identity,
  );
}

async function findAgentWorkspace(workspaceId?: string, slug?: string) {
  const where = liveWorkspaceWhere(workspaceId, slug);
  if (!where) {
    throw new AgentSignupError(
      400,
      "invalid_body",
      "Send workspaceId or slug. name must be 1 to 256 characters. permissions must be a list of path and action.",
    );
  }
  const [workspace] = await primaryDb
    .select({
      id: schema.workspaces.id,
      orgId: schema.workspaces.orgId,
      slug: schema.workspaces.slug,
    })
    .from(schema.workspaces)
    .where(where)
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
