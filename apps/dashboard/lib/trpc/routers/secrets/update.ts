import { type Secret, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { ingestAuditLogs } from "@/lib/tinybird";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const updateSecret = rateLimitedProcedure(ratelimit.update)
  .input(
    z.object({
      secretId: z.string(),
      name: z.string().optional(),
      value: z.string().optional(),
      comment: z.string().optional().nullable(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ws = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          secrets: {
            where: (table, { eq }) => eq(table.id, input.secretId),
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "We are unable to update secret. Please contact support using support@unkey.dev",
        });
      });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please contact support using support@unkey.dev.",
      });
    }
    const secret = ws.secrets.at(0);
    if (!secret) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct secrets. Please contact support using support@unkey.dev.",
      });
    }

    const update: Partial<Secret> = {};
    if (typeof input.name !== "undefined") {
      update.name = input.name;
    }

    if (typeof input.value !== "undefined") {
      // const vault = connectVault();
      // const encrypted = await vault.encrypt({
      //   keyring: ws.id,
      //   data: input.value,
      // });
      // update.encrypted = encrypted.encrypted;
      // update.encryptionKeyId = encrypted.keyId;
    }

    if (typeof input.comment !== "undefined") {
      update.comment = input.comment;
    }

    if (Object.keys(update).length === 0) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "No change detected",
      });
    }

    await db
      .update(schema.secrets)
      .set(update)
      .where(eq(schema.secrets.id, secret.id))
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the secret. Please contact support using support@unkey.dev.",
        });
      });
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
