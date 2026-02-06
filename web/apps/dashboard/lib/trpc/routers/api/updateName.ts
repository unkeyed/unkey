import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";

import { workspaceProcedure } from "../../trpc";

export const updateApiName = workspaceProcedure
  .input(
    z.object({
      name: z.string().min(3, "API names must contain at least 3 characters"),
      apiId: z.string(),
      workspaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const api = await db.query.apis
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.apiId),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update the API name. Please try again or contact support@unkey.com.",
        });
      });
    if (!api) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct API. Please try again or contact support@unkey.com.",
      });
    }
    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.apis)
          .set({
            name: input.name,
          })
          .where(eq(schema.apis.id, input.apiId))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We were unable to update the API name. Please try again or contact support@unkey.com.",
            });
          });
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "api.update",
          description: `Changed ${api.id} name from ${api.name} to ${input.name}`,
          resources: [
            {
              type: "api",
              id: api.id,
              name: input.name,
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
            "We were unable to update the API name. Please try again or contact support@unkey.com.",
        });
      });
  });
