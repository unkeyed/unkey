import { mintProxyJWT } from "@/lib/auth/proxy-jwt";
import { auth } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { createFreeWorkspaceInTx } from "@/lib/workspace/create-workspace";
import { and, eq, isNull, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { z } from "zod";
import { createClaimedWorkspace, decideRootKey, rootKeyDecisionError } from "./claim";
import { AgentSignupError } from "./errors";
import { type PermissionGrant, resolveAgentPermissions } from "./permissions";
import type { VerifiedAgent } from "./verify";

const rootKeyResponse = z.object({
  data: z.object({
    keyId: z.string().min(1),
    key: z.string().min(1),
  }),
});

export async function createAgentWorkspace(input: {
  agent: VerifiedAgent;
  name: string;
  slug: string;
  audit: { location: string; userAgent?: string };
}) {
  const now = Date.now();
  return db.transaction((tx) =>
    createClaimedWorkspace(
      async () => {
        await tx.insert(schema.agentSignups).values({
          id: newId("agentSignup"),
          agentRegistrationId: input.agent.registrationId,
          workosUserId: input.agent.userId,
          status: "pending",
          createdAtM: now,
        });
      },
      async () => {
        const created = await createFreeWorkspaceInTx(tx, {
          name: input.name,
          slug: input.slug,
          userId: input.agent.userId,
          audit: input.audit,
          createTenant: (params) => auth.createTenant(params),
          localOrgId: null,
        });
        const updated = await tx
          .update(schema.agentSignups)
          .set({
            workspaceId: created.workspaceId,
            status: "workspace_created",
            updatedAtM: Date.now(),
          })
          .where(
            and(
              eq(schema.agentSignups.agentRegistrationId, input.agent.registrationId),
              eq(schema.agentSignups.status, "pending"),
            ),
          );
        if (updated[0].affectedRows !== 1) {
          throw new Error("agent signup reservation was not updated");
        }
        return created;
      },
    ),
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

export async function issueAgentRootKey(input: {
  agent: VerifiedAgent;
  name?: string;
  permissions?: readonly PermissionGrant[];
}) {
  return db.transaction(async (tx) => {
    const [row] = await tx
      .select({
        status: schema.agentSignups.status,
        workspaceId: schema.agentSignups.workspaceId,
        rootKeyId: schema.agentSignups.rootKeyId,
        workosUserId: schema.agentSignups.workosUserId,
      })
      .from(schema.agentSignups)
      .where(eq(schema.agentSignups.agentRegistrationId, input.agent.registrationId))
      .limit(1)
      .for("update");

    const decision = decideRootKey(row, input.agent.userId);
    if (decision !== "ok" || !row?.workspaceId) {
      throw rootKeyDecisionError(decision === "ok" ? "missing" : decision);
    }

    const permissions = resolveAgentPermissions(row.workspaceId, input.permissions);
    const [workspace] = await tx
      .select({ orgId: schema.workspaces.orgId })
      .from(schema.workspaces)
      .where(eq(schema.workspaces.id, row.workspaceId))
      .limit(1);
    if (!workspace) {
      throw new Error("workspace missing for agent signup");
    }

    const created = await createRootKey({
      orgId: workspace.orgId,
      userId: input.agent.userId,
      name: input.name ?? "Agent",
      permissions,
    });
    const updated = await tx
      .update(schema.agentSignups)
      .set({
        status: "root_key_issued",
        rootKeyId: created.keyId,
        requestedPermissions: permissions,
        updatedAtM: Date.now(),
      })
      .where(
        and(
          eq(schema.agentSignups.agentRegistrationId, input.agent.registrationId),
          eq(schema.agentSignups.status, "workspace_created"),
          isNull(schema.agentSignups.rootKeyId),
        ),
      );
    if (updated[0].affectedRows !== 1) {
      throw new AgentSignupError(
        409,
        "root_key_exists",
        "This agent registration already received its first root key.",
      );
    }
    return {
      keyId: created.keyId,
      key: created.key,
      permissions,
    };
  });
}
