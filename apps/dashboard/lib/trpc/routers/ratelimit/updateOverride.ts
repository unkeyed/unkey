import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { workspaceProcedure } from "../../trpc";

export const updateOverride = workspaceProcedure
  .input(
    z.object({
      id: z.string(),
      limit: z.number(),
      duration: z.number(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const override = await db.query.ratelimitOverrides
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.id),
            isNull(table.deletedAtM),
          ),
        with: {
          namespace: {
            columns: {
              id: true,
              name: true,
            },
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the override for this namespace. Please try again or contact support@unkey.dev",
        });
      });

    if (!override) {
      throw new TRPCError({
        message:
          "We are unable to find the correct override. Please try again or contact support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.ratelimitOverrides)
          .set({
            limit: input.limit,
            duration: input.duration,
            updatedAtM: Date.now(),
            async: false,
          })
          .where(eq(schema.ratelimitOverrides.id, override.id))
          .catch((_err) => {
            throw new TRPCError({
              message:
                "We are unable to update the override for this namespace. Please try again or contact support@unkey.dev.",
              code: "INTERNAL_SERVER_ERROR",
            });
          });
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "ratelimit.set_override",
          description: `Changed ${override.id} limits from ${override.limit}/${override.duration} to ${input.limit}/${input.duration}`,
          resources: [
            {
              type: "ratelimitNamespace",
              id: override.namespace.id,
              name: override.namespace.name,
            },
            {
              type: "ratelimitOverride",
              id: override.id,
              name: override.identifier,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        }).catch((_err) => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to update the override. Please try again or contact support@unkey.dev",
          });
        });
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the override for this namespace. Please try again or contact support@unkey.dev",
        });
      });
  });
