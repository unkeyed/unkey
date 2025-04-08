import { insertAuditLogs } from "@/lib/audit";
import { auth as authProvider } from "@/lib/auth/server";
import { type Workspace, db, schema } from "@/lib/db";
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

    const orgId = await authProvider.createTenant({
      name: input.name,
      userId,
    });

    const workspace: Workspace = {
      id: newId("workspace"),
      orgId: orgId,
      // dumb hack to keep the unique property but also clearly mark it as a workos identifier
      clerkTenantId: `workos_${orgId}`,
      name: input.name,
      plan: "free",
      tier: "Free",
      stripeCustomerId: null,
      stripeSubscriptionId: null,
      trialEnds: new Date(Date.now() + 1000 * 60 * 60 * 24 * 14), // 2 weeks
      features: {},
      betaFeatures: {},
      planLockedUntil: null,
      planChanged: null,
      subscriptions: {},
      planDowngradeRequest: null,
      enabled: true,
      deleteProtection: true,
      createdAtM: Date.now(),
      updatedAtM: null,
      deletedAtM: null,
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
