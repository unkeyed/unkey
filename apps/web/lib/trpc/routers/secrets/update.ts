import { type Secret, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { AesGCM, getEncryptionKeyFromEnv } from "@unkey/encryption";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const updateSecret = t.procedure
  .use(auth)
  .input(
    z.object({
      secretId: z.string(),
      name: z.string().optional(),
      value: z.string().optional(),
      comment: z.string().optional().nullable(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      with: {
        secrets: {
          where: (table, { eq, and, isNull }) =>
            and(eq(table.id, input.secretId), isNull(table.deletedAt)),
        },
      },
    });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }
    const secret = ws.secrets.at(0);
    if (!secret) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "secret not found",
      });
    }

    const update: Partial<Secret> = {};
    if (typeof input.name !== "undefined") {
      update.name = input.name;
    }

    if (typeof input.value !== "undefined") {
      const encryptionKey = getEncryptionKeyFromEnv(env());
      if (encryptionKey.err) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "missing encryption key in env",
        });
      }

      const aes = await AesGCM.withBase64Key(encryptionKey.val.key);

      const { iv, ciphertext } = await aes.encrypt(input.value);

      update.iv = iv;
      update.ciphertext = ciphertext;
      update.encryptionKeyVersion = encryptionKey.val.version;
    }

    if (typeof input.comment !== "undefined") {
      update.comment = input.comment;
    }

    if (Object.keys(update).length === 0) {
      throw new TRPCError({ code: "PRECONDITION_FAILED", message: "No change detected" });
    }

    await db.update(schema.secrets).set(update).where(eq(schema.secrets.id, secret.id));

    await ingestAuditLogs({
      workspaceId: ws.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "secret.update",
      description: `Updated ${secret.id}`,
      resources: [
        {
          type: "secret",
          id: secret.id,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return;
  });
