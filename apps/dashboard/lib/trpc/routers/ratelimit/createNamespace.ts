import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { DatabaseError } from "@planetscale/database";
import { newId } from "@unkey/id";
import { auth, t } from "../../trpc";
export const createNamespace = t.procedure
  .use(auth)
  .input(
    z.object({
      name: z.string().min(1).max(50),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ws = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create a new namespace. Please try again or contact support@unkey.dev",
        });
      });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please try again or contact support@unkey.dev.",
      });
    }

    const namespaceId = newId("ratelimitNamespace");
    await db
      .transaction(async (tx) => {
        await tx.insert(schema.ratelimitNamespaces).values({
          id: namespaceId,
          name: input.name,
          workspaceId: ws.id,

          createdAt: new Date(),
        });
        await insertAuditLogs(tx, {
          workspaceId: ws.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "ratelimitNamespace.create",
          description: `Created ${namespaceId}`,
          resources: [
            {
              type: "ratelimitNamespace",
              id: namespaceId,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })
      .catch((e) => {
        if (e instanceof DatabaseError && e.body.message.includes("desc = Duplicate entry")) {
          throw new TRPCError({
            code: "CONFLICT",
            message: "duplicate namespace name. Please use a unique name for each namespace.",
          });
        }
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create namespace. Please try again or contact support@unkey.dev",
        });
      });

    return {
      id: namespaceId,
    };
  });
