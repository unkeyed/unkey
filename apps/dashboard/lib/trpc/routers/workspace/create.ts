import { insertAuditLogs } from "@/lib/audit";
import { type Workspace, db, schema } from "@/lib/db";
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
        message:
          "We are not able to authenticate the user. Please make sure you are logged in and try again",
      });
    }

    const subscriptions = defaultProSubscriptions();

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
      subscriptions,
      createdAt: new Date(),
      deletedAt: null,
      planDowngradeRequest: null,
      enabled: true,
      deleteProtection: true,
    };
    await db
      .transaction(async (tx) => {
        await tx.insert(schema.workspaces).values(workspace);

        const auditLogBucketId = newId("auditLogBucket");
        await tx.insert(schema.auditLogBucket).values({
          id: auditLogBucketId,
          workspaceId: workspace.id,
          name: "unkey_mutations",
          deleteProtection: true,
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
          {
            workspaceId: workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "auditLogBucket.create",
            description: `Created ${auditLogBucketId}`,
            resources: [
              {
                type: "auditLogBucket",
                id: auditLogBucketId,
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
      organizationId: org.id,
    };
  });
