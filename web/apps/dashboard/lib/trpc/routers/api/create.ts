import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const createApi = workspaceProcedure
  .input(
    z.object({
      name: z
        .string()
        .min(3, "API names must contain at least 3 characters")
        .max(50, "API names must contain at most 50 characters"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    try {
      return await db.transaction(async (tx) => {
        const result = await createApiCore(input, ctx, tx);
        return { id: result.id };
      });
    } catch (_err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "We are unable to create the API. Please try again or contact support@unkey.dev",
      });
    }
  });

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

export async function createApiCore(
  input: CreateApiInput,
  ctx: CreateApiContext,
  tx: DatabaseTransaction,
) {
  const keyAuthId = newId("keyAuth");
  const apiId = newId("api");

  await tx.insert(schema.keyAuth).values({
    id: keyAuthId,
    workspaceId: ctx.workspace.id,
    createdAtM: Date.now(),
  });

  await tx.insert(schema.apis).values({
    id: apiId,
    name: input.name,
    workspaceId: ctx.workspace.id,
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
