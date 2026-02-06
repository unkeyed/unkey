import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { workspaceProcedure } from "../../trpc";

export const updateNamespaceName = workspaceProcedure
  .input(
    z.object({
      name: z.string().min(3, "namespace names must contain at least 3 characters"),
      namespaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const namespace = await db.query.ratelimitNamespaces
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            isNull(table.deletedAtM),
            eq(table.id, input.namespaceId),
          ),
      })

      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the name for this namespace. Please try again or contact support@unkey.com",
        });
      });

    if (!namespace) {
      throw new TRPCError({
        message:
          "We are unable to find the correct workspace. Please try again or contact support@unkey.com",
        code: "NOT_FOUND",
      });
    }

    await db.transaction(async (tx) => {
      await tx
        .update(schema.ratelimitNamespaces)
        .set({
          name: input.name,
        })
        .where(eq(schema.ratelimitNamespaces.id, input.namespaceId))
        .catch((_err) => {
          throw new TRPCError({
            message:
              "We are unable to update the namespace name. Please try again or contact support@unkey.com",
            code: "INTERNAL_SERVER_ERROR",
          });
        });
      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        event: "ratelimitNamespace.update",
        description: `Changed ${namespace.id} name from ${namespace.name} to ${input.name}`,
        resources: [
          {
            type: "ratelimitNamespace",
            id: namespace.id,
            name: input.name,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      }).catch((_err) => {
        throw new TRPCError({
          message:
            "We are unable to update the namespace name. Please try again or contact support@unkey.com",
          code: "INTERNAL_SERVER_ERROR",
        });
      });
    });
  });
