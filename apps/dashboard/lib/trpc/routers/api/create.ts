import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { auth, t } from "../../trpc";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { newId } from "@unkey/id";

export const createApi = t.procedure
  .use(auth)
  .input(
    z.object({
      name: z
        .string()
        .min(3, "workspace names must contain at least 3 characters")
        .max(50, "workspace names must contain at most 50 characters"),
    })
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
            "We are unable to create an API. Please try again or contact support@unkey.dev",
        });
      });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "The workspace does not exist.",
      });
    }

    const keyAuthId = newId("keyAuth");
    try {
      await db.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: ws.id,
        createdAt: new Date(),
      });
    } catch (_err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "We are unable to create an API. Please try again or contact support@unkey.dev",
      });
    }

    const apiId = newId("api");

    await db
      .transaction(async (tx) => {
        await tx
          .insert(schema.apis)
          .values({
            id: apiId,
            name: input.name,
            workspaceId: ws.id,
            keyAuthId,
            authType: "key",
            ipWhitelist: null,
            createdAt: new Date(),
          })
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to create the API. Please try again or contact support@unkey.dev",
            });
          });

        await insertAuditLogs(tx, {
          workspaceId: ws.id,
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
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create the API. Please try again or contact support@unkey.dev",
        });
      });

    return {
      id: apiId,
    };
  });
