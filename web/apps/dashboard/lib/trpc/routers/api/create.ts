import { insertAuditLogs } from "@/lib/audit";
import { type db, schema } from "@/lib/db";
import { ensureDefaultProjectId } from "@/lib/projects/ensure-default-project-id";
import { newId } from "@unkey/id";

export async function createApiCore(
  input: CreateApiInput,
  ctx: CreateApiContext,
  tx: DatabaseTransaction,
) {
  const projectId = await ensureDefaultProjectId(tx, ctx.workspace.id);
  const keyAuthId = newId("keyAuth");
  const apiId = newId("api");

  await tx.insert(schema.keyAuth).values({
    id: keyAuthId,
    workspaceId: ctx.workspace.id,
    projectId,
    createdAtM: Date.now(),
  });

  await tx.insert(schema.apis).values({
    id: apiId,
    name: input.name,
    workspaceId: ctx.workspace.id,
    projectId,
    keyAuthId,
    authType: "key",
    ipWhitelist: null,
    createdAtM: Date.now(),
  });

  await insertAuditLogs(tx, {
    workspaceId: ctx.workspace.id,
    actor: {
      type: "user",
      id: ctx.user.id,
    },
    event: "api.create",
    description: `Created ${apiId}`,
    resources: [
      {
        type: "api",
        id: apiId,
        name: input.name,
      },
    ],
    context: {
      location: ctx.audit.location,
      userAgent: ctx.audit.userAgent,
    },
  });

  return {
    id: apiId,
    keyAuthId,
  };
}

type CreateApiInput = {
  name: string;
};

type CreateApiContext = {
  workspace: { id: string };
  user: { id: string };
  audit: {
    location: string;
    userAgent?: string;
  };
};

type DatabaseTransaction = Parameters<Parameters<typeof db.transaction>[0]>[0];
