import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { newId } from "@unkey/id";
import { auth, t } from "../../trpc";

export const createApi = t.procedure
.use(auth)
  .input(
    z.object({
      name: z
        .string()
        .min(3, "API names must contain at least 3 characters")
        .max(50, "API names must contain at most 50 characters"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const keyAuthId = newId("keyAuth");
    try {
      await db.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: ctx.workspace.id,
        createdAtM: Date.now(),
      });
    } catch (_err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "We are unable to create an API. Please try again or contact support@unkey.dev",
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
            workspaceId: ctx.workspace.id,
            keyAuthId,
            authType: "key",
            ipWhitelist: null,
            createdAtM: Date.now(),
          })
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to create the API. Please try again or contact support@unkey.dev",
            });
          });

        await insertAuditLogs(tx, ctx.workspace.auditLogBucket.id, {
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
          message: "We are unable to create the API. Please try again or contact support@unkey.dev",
        });
      });

    return {
      id: apiId,
    };
  });
