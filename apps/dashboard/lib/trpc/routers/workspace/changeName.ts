import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogsTinybird } from "@/lib/tinybird";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { clerkClient } from "@clerk/nextjs";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const changeWorkspaceName = rateLimitedProcedure(ratelimit.update)
  .input(
    z.object({
      name: z.string().min(3, "workspace names must contain at least 3 characters"),
      workspaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const ws = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.id, input.workspaceId), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the workspace name. Please contact support using support@unkey.dev",
        });
      });
    if (!ws || ws.tenantId !== ctx.tenant.id) {
      throw new Error("Workspace not found, Please sign back in and try again");
    }
    await db.transaction(async (tx) => {
      await tx
        .update(schema.workspaces)
        .set({
          name: input.name,
        })
        .where(eq(schema.workspaces.id, input.workspaceId))
        .catch((_err) => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to update the workspace name. Please contact support using support@unkey.dev",
          });
        });
      await insertAuditLogs(tx, {
        workspaceId: ws.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: `Changed name from ${ws.name} to ${input.name}`,
        resources: [
          {
            type: "workspace",
            id: ws.id,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
      await ingestAuditLogsTinybird({
        workspaceId: ws.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: `Changed name from ${ws.name} to ${input.name}`,
        resources: [
          {
            type: "workspace",
            id: ws.id,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
      if (ctx.tenant.id.startsWith("org_")) {
        await clerkClient.organizations.updateOrganization(ctx.tenant.id, {
          name: input.name,
        });
      }
    });
  });
