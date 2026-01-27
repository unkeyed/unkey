import { insertAuditLogs } from "@/lib/audit";
import { auth as authProvider } from "@/lib/auth/server";
import { type Workspace, db, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { freeTierQuotas } from "@/lib/quotas";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { protectedProcedure } from "../../trpc";

export const createWorkspace = protectedProcedure
  .input(
    z.object({
      name: z.string().min(3).max(50),
      slug: z.string().regex(/^(?!-)[a-z0-9]+(?:-[a-z0-9]+)*(?<!-)$/, {
        error: "Use lowercase letters, numbers, and hyphens (no leading/trailing hyphens).",
      }),
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

    const orgId = await db
      .transaction(async (tx) => {
        if (env().AUTH_PROVIDER === "local") {
          // Check if this user already has a workspace
          const existingWorkspaces = await tx.query.workspaces.findMany({
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

        const duplicateSlug = await tx.query.workspaces.findFirst({
          where: (workspaces, { eq }) => eq(workspaces.slug, input.slug),
        });

        if (duplicateSlug) {
          throw new TRPCError({
            code: "CONFLICT",
            message: "A workspace with this slug already exists.",
          });
        }

        const orgId = await authProvider.createTenant({
          name: input.name,
          userId,
        });

        const workspace: Workspace = {
          id: newId("workspace"),
          orgId: orgId,
          name: input.name,
          slug: input.slug,
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
          k8sNamespace: null,
          paymentFailedAt : null,
          paymentFailureNotifiedAt: null,
          subscriptionStatus: null
        };

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

        // Invalidate workspace cache for the new orgId and current user's orgId
        // The new orgId needs invalidation for consistency (though it's new)
        // The current user's orgId needs invalidation since they're switching away
        await invalidateWorkspaceCache(orgId);
        await invalidateWorkspaceCache(ctx.tenant.id);
        return orgId;
      })
      .catch((err) => {
        if (err instanceof TRPCError) {
          throw err;
        }
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create the workspace. Please try again or contact support@unkey.dev",
        });
      });

    return {
      orgId,
    };
  });
