import { db, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { ingestAuditLogs } from "@/lib/tinybird";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { DatabaseError } from "@planetscale/database";
import { TRPCError } from "@trpc/server";
import { AesGCM } from "@unkey/encryption";
import { newId } from "@unkey/id";
import { z } from "zod";

export const createSecret = rateLimitedProcedure(ratelimit.create)
  .input(
    z.object({
      name: z.string(),
      value: z.string(),
      comment: z.string().optional(),
    }),
  )
  .mutation(async ({ ctx }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please contact support using support@unkey.dev.",
      });
    }

    // const vault = connectVault();

    // const encrypted = await vault.encrypt({
    //   keyring: ws.id,
    //   data: input.value,
    // });

    const secretId = newId("secret");
    // await db
    //   .insert(schema.secrets)
    //   .values({
    //     id: secretId,
    //     comment: input.comment,
    //     name: input.name,
    //     workspaceId: ws.id,
    //     encrypted: encrypted.encrypted,
    //     encryptionKeyId: encrypted.keyId,
    //   })
    //   .catch((err) => {
    //     if (err instanceof DatabaseError && err.body.message.includes("desc = Duplicate entry")) {
    //       throw new TRPCError({
    //         code: "PRECONDITION_FAILED",
    //         message: "Secrets must have unique names",
    //       });
    //     }
    //     throw err;
    //   });

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
