import { insertAuditLogs } from "@/lib/audit";
import type { CreateTenantParams } from "@/lib/auth/types";
import { type InsertWorkspace, primaryDb } from "@/lib/db";
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
  deleteTenant: (orgId: string) => Promise<void>;
  localOrgId: string | null;
};

export type WorkspaceCreateDeps = {
  findWorkspacesByOrgId: (orgId: string) => Promise<Array<{ id: string }>>;
  findWorkspaceBySlug: (slug: string) => Promise<{ id: string } | undefined>;
  persist: (input: PersistFreeWorkspaceInput) => Promise<CreatedWorkspace>;
};

type PersistFreeWorkspaceInput = {
  orgId: string;
  name: string;
  slug: string;
  userId: string;
  audit: CreateFreeWorkspaceInput["audit"];
};

export type CreatedWorkspace = {
  orgId: string;
  workspaceId: string;
  slug: string;
};

function productionWorkspaceCreateDeps(): WorkspaceCreateDeps {
  return {
    findWorkspacesByOrgId(orgId) {
      return primaryDb.query.workspaces.findMany({
        where: (workspaces, { eq }) => eq(workspaces.orgId, orgId),
        columns: { id: true },
      });
    },
    findWorkspaceBySlug(slug) {
      return primaryDb.query.workspaces.findFirst({
        where: (workspaces, { eq }) => eq(workspaces.slug, slug),
        columns: { id: true },
      });
    },
    persist(input) {
      return primaryDb.transaction((tx) => insertFreeWorkspace(tx, input));
    },
  };
}

export async function createFreeWorkspace(
  input: CreateFreeWorkspaceInput,
  deps: WorkspaceCreateDeps = productionWorkspaceCreateDeps(),
): Promise<CreatedWorkspace> {
  if (!input.userId) {
    throw new WorkspaceCreateError(
      "UNAUTHORIZED",
      "We are not able to authenticate the user. Please make sure you are logged in and try again",
    );
  }

  const localOrgId = input.localOrgId;
  if (localOrgId) {
    const existingWorkspaces = await deps.findWorkspacesByOrgId(localOrgId);
    if (existingWorkspaces.length > 0) {
      throw new WorkspaceCreateError(
        "METHOD_NOT_SUPPORTED",
        "You cannot create additional workspaces in local development mode. Use workOS auth provider if you need to test multi-workspace functionality.",
      );
    }
  }

  const duplicateSlug = await deps.findWorkspaceBySlug(input.slug);
  if (duplicateSlug) {
    throw new WorkspaceCreateError("CONFLICT", "A workspace with this slug already exists.");
  }

  const orgId = await input.createTenant({
    name: input.name,
    userId: input.userId,
    ...(input.metadata ? { metadata: input.metadata } : {}),
  });

  try {
    return await deps.persist({
      orgId,
      name: input.name,
      slug: input.slug,
      userId: input.userId,
      audit: input.audit,
    });
  } catch (error) {
    try {
      await input.deleteTenant(orgId);
    } catch (cleanupError) {
      console.error(
        "failed to delete organization after workspace create failed",
        cleanupError instanceof Error ? cleanupError.name : "unknown",
      );
    }
    throw error;
  }
}

async function insertFreeWorkspace(
  tx: Transaction,
  input: PersistFreeWorkspaceInput,
): Promise<CreatedWorkspace> {
  const workspace: InsertWorkspace = {
    id: newId("workspace"),
    orgId: input.orgId,
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
    orgId: input.orgId,
    workspaceId: workspace.id,
    slug: input.slug,
  };
}
