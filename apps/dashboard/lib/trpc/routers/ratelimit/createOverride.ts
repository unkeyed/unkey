import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, schema, sql } from "@/lib/db";
import { newId } from "@unkey/id";
import { requireUser, requireWorkspace, t } from "../../trpc";
export const createOverride = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      namespaceId: z.string(),
      identifier: z.string(),
      limit: z.number(),
      duration: z.number(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const namespace = await db.query.ratelimitNamespaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.namespaceId),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create an override for this namespace. Please try again or contact support@unkey.dev",
        });
      });
    if (!namespace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct namespace. Please try again or contact support@unkey.dev.",
      });
    }

    const id = newId("ratelimitOverride");
    await db
      .transaction(async (tx) => {
        const existing = await tx.query.ratelimitOverrides.findFirst({
          where: (table, { and, eq }) =>
            and(eq(table.namespaceId, namespace.id), eq(table.identifier, input.identifier)),
        });

        if (existing) {
          await tx
            .update(schema.ratelimitOverrides)
            .set({
              limit: input.limit,
              duration: input.duration,
              updatedAtM: Date.now(),
              deletedAtM: null,
            })
            .where(sql`namespace_id = ${namespace.id} AND identifier = ${input.identifier}`);
        } else {
          await tx.insert(schema.ratelimitOverrides).values({
            id,
            workspaceId: ctx.workspace.id,
            namespaceId: namespace.id,
            identifier: input.identifier,
            limit: input.limit,
            duration: input.duration,
            async: false,
            createdAtM: Date.now(),
          });
        }
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: existing ? "ratelimit.update" : "ratelimit.set_override",
          description: existing ? `Updated ${input.identifier}` : `Created ${input.identifier}`,
          resources: [
            {
              type: "ratelimitNamespace",
              id: input.namespaceId,
              name: namespace.name,
            },
            {
              type: "ratelimitOverride",
              id: existing ? existing.id : id,
              name: input.identifier,
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
            "We are unable to create the override. Please try again or contact support@unkey.dev",
        });
      });

    return {
      id,
    };
  });
