import { db } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
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
          where: (table, { eq }) => eq(table.id, input.secretId),
        },
      },
    });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Sorry, we are unable to find the correct workspace, please contact support using support@unkey.dev.",
      });
    }
    const secret = ws.secrets.at(0);
    if (!secret) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Sorry, we are unable to find the correct secrets, please contact support using support@unkey.dev.",
      });
    }

    // const vault = connectVault();
    // const decrypted = await vault.decrypt({
    //   keyring: ws.id,
    //   encrypted: secret.encrypted,
    // });

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
      value: "", //decrypted.plaintext,
    };
  });
