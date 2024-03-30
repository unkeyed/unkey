import { db, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { ingestAuditLogs } from "@/lib/tinybird";
import { DatabaseError } from "@planetscale/database";
import { TRPCError } from "@trpc/server";
import { AesGCM, getEncryptionKeyFromEnv } from "@unkey/encryption";
import { newId } from "@unkey/id";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const createSecret = t.procedure
  .use(auth)
  .input(
    z.object({
      name: z.string(),
      value: z.string(),
      comment: z.string().optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }

    const { key, version } = getEncryptionKeyFromEnv(env());

    const aes = await AesGCM.withBase64Key(key);

    const { iv, ciphertext } = await aes.encrypt(input.value);

    const secretId = newId("secret");
    await db
      .insert(schema.secrets)
      .values({
        id: secretId,
        algorithm: AesGCM.algorithm,
        ciphertext,
        iv,
        comment: input.comment,
        name: input.name,
        workspaceId: ws.id,
        encryptionKeyVersion: version,
      })
      .catch((err) => {
        if (err instanceof DatabaseError && err.body.message.includes("desc = Duplicate entry")) {
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: "Secrets must have unique names",
          });
        }
        throw err;
      });

    await ingestAuditLogs({
      workspaceId: ws.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "secret.create",
      description: `Created ${secretId}`,
      resources: [
        {
          type: "secret",
          id: secretId,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return {
      secretId,
    };
  });
