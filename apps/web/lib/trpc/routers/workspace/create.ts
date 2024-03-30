import { type Workspace, db, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { clerkClient } from "@clerk/nextjs";
import { TRPCError } from "@trpc/server";
import { defaultProSubscriptions } from "@unkey/billing";
import { newId } from "@unkey/id";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const createWorkspace = t.procedure
  .use(auth)
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
        message: "unable to find userId",
      });
    }

    const org = await clerkClient.organizations.createOrganization({
      name: input.name,
      createdBy: userId,
    });

    const workspace: Workspace = {
      id: newId("workspace"),
      tenantId: org.id,
      name: input.name,
      plan: "pro",
      stripeCustomerId: null,
      stripeSubscriptionId: null,
      trialEnds: new Date(Date.now() + 1000 * 60 * 60 * 24 * 14), // 2 weeks
      features: {},
      betaFeatures: {},
      planLockedUntil: null,
      planChanged: null,
      subscriptions: defaultProSubscriptions(),
      createdAt: new Date(),
      deletedAt: null,
      planDowngradeRequest: null,
      enabled: true,
    };
    await db.insert(schema.workspaces).values(workspace);
    await ingestAuditLogs({
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
    });

    return {
      workspace,
      organizationId: org.id,
    };
  });
