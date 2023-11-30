import { QUOTA } from "@/lib/constants/quotas";
import { Workspace, db, schema } from "@/lib/db";
import { clerkClient } from "@clerk/nextjs";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { auth, t } from "../trpc";

export const workspaceRouter = t.router({
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        name: z.string().min(1).max(50),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const userId = ctx.user?.id;
      if (!userId) {
        throw new TRPCError({ code: "UNAUTHORIZED", message: "unable to find userId" });
      }

      const org = await clerkClient.organizations.createOrganization({
        name: input.name,
        createdBy: userId,
      });

      const workspace: Workspace = {
        id: newId("workspace"),
        slug: null,
        tenantId: org.id,
        name: input.name,
        plan: "pro",
        stripeCustomerId: null,
        stripeSubscriptionId: null,
        maxActiveKeys: QUOTA.pro.maxActiveKeys,
        maxVerifications: QUOTA.pro.maxVerifications,
        usageActiveKeys: null,
        usageVerifications: null,
        lastUsageUpdate: null,
        billingPeriodStart: null,
        billingPeriodEnd: null,
        trialEnds: new Date(Date.now() + 1000 * 60 * 60 * 24 * 14), // 2 weeks
        features: {},
        betaFeatures: {},
      };
      await db.insert(schema.workspaces).values(workspace);

      return {
        workspace,
        organizationId: org.id,
      };
    }),
});
