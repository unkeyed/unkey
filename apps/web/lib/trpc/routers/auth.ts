import { auth } from "@/lib/auth";
import { newId } from "@unkey/id";
import { z } from "zod";
import { authenticateUser, t } from "../trpc";

export const authRouter = t.router({
  createWorkspace: t.procedure
    .use(authenticateUser)
    .input(
      z.object({
        name: z.string(),
        slug: z.string(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      await auth.createWorkspace(ctx.user.id, {
        id: newId("workspace"),
        name: input.name,
        slug: input.slug,
        createdAt: new Date(),
        features: {},
        betaFeatures: {},
        plan: "free",
        deletedAt: null,
        planChanged: null,
        stripeCustomerId: null,
        stripeSubscriptionId: null,
        trialEnds: null,
        planDowngradeRequest: null,
        planLockedUntil: null,
        subscriptions: null,
      });
      return;
    }),
});
