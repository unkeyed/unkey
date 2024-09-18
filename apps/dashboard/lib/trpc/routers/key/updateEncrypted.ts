import { db, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const updateKeyEncrypted = t.procedure
  .use(auth)
  .input(
    z.object({
      keyId: z.string(),
      encrypted: z.string(),
      encryptiodKeyId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const key = await db.query.keys.findFirst({
      where: (table, { eq, isNull, and }) =>
        and(eq(table.id, input.keyId), isNull(table.deletedAt)),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please contact support using support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    const tuple = {
      keyId: input.keyId,
      encrypted: input.encrypted,
      encryptionKeyId: input.encryptiodKeyId,
      workspaceId: ctx.tenant.id,
    };
    await db
      .insert(schema.encryptedKeys)
      .values({ ...tuple })
      .onDuplicateKeyUpdate({ set: { ...tuple } })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update key encrypted. Please contact support using support@unkey.dev",
        });
      });

    await ingestAuditLogs({
      workspaceId: key.workspace.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "key.update",
      description: `Created a encrypted relation to ${key.id}`,
      resources: [
        {
          type: "key",
          id: key.id,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
    return true;
  });
