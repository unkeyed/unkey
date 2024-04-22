import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { AesGCM, getDecryptionKeyFromEnv } from "@unkey/encryption";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const decryptSecret = t.procedure
  .use(auth)
  .input(
    z.object({
      secretId: z.string(),
    }),
  )
  .output(
    z.object({
      value: z.string(),
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

    const decryptionKey = getDecryptionKeyFromEnv(env(), secret.encryptionKeyVersion);
    if (decryptionKey.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "missing encryption key in env",
      });
    }
    const aes = await AesGCM.withBase64Key(decryptionKey.val);

    const value = await aes.decrypt({ iv: secret.iv, ciphertext: secret.ciphertext });

    await ingestAuditLogs({
      workspaceId: ws.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "secret.decrypt",
      description: `Decrypted ${secret.id}`,
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

    return {
      value,
    };
  });
