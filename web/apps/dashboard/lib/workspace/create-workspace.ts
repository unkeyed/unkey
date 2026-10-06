import { insertAuditLogs } from "@/lib/audit";
import type { CreateTenantParams } from "@/lib/auth/types";
import type { InsertWorkspace } from "@/lib/db";
import { freeTierLimits } from "@/lib/limits";
import { type Transaction, isDuplicateKeyError, schema } from "@unkey/db";
import { dns1035, newId } from "@unkey/id";

export type WorkspaceCreateCode = "UNAUTHORIZED" | "CONFLICT" | "METHOD_NOT_SUPPORTED";

export class WorkspaceCreateError extends Error {
  readonly code: WorkspaceCreateCode;

  constructor(code: WorkspaceCreateCode, message: string) {
    super(message);
    this.name = "WorkspaceCreateError";
    this.code = code;
  }
}

export type CreateFreeWorkspaceInput = {
  name: string;
  slug: string;
  userId: string;
  audit: {
    location: string;
    userAgent?: string;
  };
  metadata?: Record<string, string>;
  createTenant: (params: CreateTenantParams) => Promise<string>;
  localOrgId: string | null;
};

export type CreatedWorkspace = {
  orgId: string;
  workspaceId: string;
  slug: string;
};

export async function createFreeWorkspaceInTx(
  tx: Transaction,
  input: CreateFreeWorkspaceInput,
): Promise<CreatedWorkspace> {
  if (!input.userId) {
    throw new WorkspaceCreateError(
      "UNAUTHORIZED",
      "We are not able to authenticate the user. Please make sure you are logged in and try again",
    );
  }

  const localOrgId = input.localOrgId;
  if (localOrgId) {
    const existingWorkspaces = await tx.query.workspaces.findMany({
      where: (workspaces, { eq }) => eq(workspaces.orgId, localOrgId),
      columns: { id: true },
    });

    if (existingWorkspaces.length > 0) {
      throw new WorkspaceCreateError(
        "METHOD_NOT_SUPPORTED",
        "You cannot create additional workspaces in local development mode. Use workOS auth provider if you need to test multi-workspace functionality.",
      );
    }
  }

  const duplicateSlug = await tx.query.workspaces.findFirst({
    where: (workspaces, { eq }) => eq(workspaces.slug, input.slug),
    columns: { id: true },
  });

  if (duplicateSlug) {
    throw new WorkspaceCreateError("CONFLICT", "A workspace with this slug already exists.");
  }

  const orgId = await input.createTenant({
    name: input.name,
    userId: input.userId,
    ...(input.metadata ? { metadata: input.metadata } : {}),
  });

  const workspace: InsertWorkspace = {
    id: newId("workspace"),
    orgId,
    name: input.name,
    slug: input.slug,
    betaFeatures: {},
    enabled: true,
    deleteProtection: true,
    createdAtM: Date.now(),
    updatedAtM: null,
    deletedAtM: null,
    k8sNamespace: dns1035(),
  };

  try {
    await tx.insert(schema.workspaces).values(workspace);
  } catch (error) {
    if (isDuplicateKeyError(error)) {
      throw new WorkspaceCreateError("CONFLICT", "A workspace with this slug already exists.");
    }
    throw error;
  }

  await tx.insert(schema.limits).values({
    workspaceId: workspace.id,
    ...freeTierLimits,
  });
  await tx.insert(schema.workspaceBilling).values({
    workspaceId: workspace.id,
    tier: "Free",
  });

  await insertAuditLogs(tx, [
    {
      workspaceId: workspace.id,
      actor: { type: "user", id: input.userId },
      event: "workspace.create",
      description: `Created ${workspace.id}`,
      resources: [
        {
          type: "workspace",
          id: workspace.id,
          name: input.name,
        },
      ],
      context: {
        location: input.audit.location,
        userAgent: input.audit.userAgent,
      },
    },
  ]);

  return {
    orgId,
    workspaceId: workspace.id,
    slug: input.slug,
  };
}
