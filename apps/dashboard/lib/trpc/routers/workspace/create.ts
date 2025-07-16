import { insertAuditLogs } from "@/lib/audit";
import { auth as authProvider } from "@/lib/auth/server";
import { type Workspace, db, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { freeTierQuotas } from "@/lib/quotas";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { requireUser, t } from "../../trpc";

export const createWorkspace = t.procedure
  .use(requireUser)
  .input(
    z.object({
      name: z.string().min(1).max(50),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const userId = ctx.user?.id;

    if (!userId) {
      throw new TRPCError({
        code: "UNAUTHORIZED",
        message:
          "We are not able to authenticate the user. Please make sure you are logged in and try again",
      });
    }

    if (env().AUTH_PROVIDER === "local") {
      // Check if this user already has a workspace
      const existingWorkspaces = await db.query.workspaces.findMany({
        where: (workspaces, { eq }) => eq(workspaces.orgId, ctx.tenant.id),
      });

      if (existingWorkspaces.length > 0) {
        throw new TRPCError({
          code: "METHOD_NOT_SUPPORTED",
          message:
            "You cannot create additional workspaces in local development mode. Use workOS auth provider if you need to test multi-workspace functionality.",
        });
      }
    }

    const orgId = await authProvider.createTenant({
      name: input.name,
      userId,
    });

    const workspace: Workspace = {
      id: newId("workspace"),
      orgId: orgId,
      name: input.name,
      plan: "free",
      tier: "Free",
      stripeCustomerId: null,
      stripeSubscriptionId: null,
      features: {},
      betaFeatures: {},
      subscriptions: {},
      enabled: true,
      deleteProtection: true,
      createdAtM: Date.now(),
      updatedAtM: null,
      deletedAtM: null,
      partitionId: null,
    };

    await db
      .transaction(async (tx) => {
        await tx.insert(schema.workspaces).values(workspace);
        await tx.insert(schema.quotas).values({
          workspaceId: workspace.id,
          ...freeTierQuotas,
        });

        await insertAuditLogs(tx, [
          {
            workspaceId: workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "workspace.create",
            description: `Created ${workspace.id}`,
            resources: [
              {
                type: "workspace",
                id: workspace.id,
                name: input.name,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          },
        ]);
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create the workspace. Please try again or contact support@unkey.dev",
        });
      });

    return {
      workspace,
      organizationId: orgId,
    };
  });
