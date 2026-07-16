import { getStripeClient } from "@/lib/stripe";
import { linkDeploySubscription as linkDeploySubscriptionCore } from "@/lib/stripe/linkDeploySubscription";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, requireWorkspaceAdmin, withRatelimit, workspaceProcedure } from "../../trpc";

/**
 * Persists a subscription-mode Compute checkout onto the caller's workspace,
 * called by /success when the user returns from Stripe. The heavy lifting
 * (ownership + payment verification, plan detection, idempotent write) lives in
 * the shared linker so the checkout.session.completed webhook can reuse it; this
 * wrapper only supplies the workspace/actor and maps failures to TRPCErrors.
 *
 * Admin-guarded (writing billing state) and rate-limited (it makes two Stripe
 * calls per invocation).
 */
export const linkDeploySubscription = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .use(withRatelimit(ratelimit.update))
  .input(z.object({ sessionId: z.string().min(1) }))
  .mutation(async ({ ctx, input }) => {
    const stripe = getStripeClient();

    const result = await linkDeploySubscriptionCore(stripe, {
      sessionId: input.sessionId,
      expectedWorkspaceId: ctx.workspace.id,
      audit: {
        actor: { type: "user", id: ctx.user.id },
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    if (!result.ok) {
      throw new TRPCError({
        code:
          result.reason === "forbidden"
            ? "FORBIDDEN"
            : result.reason === "session_not_found" || result.reason === "workspace_not_found"
              ? "NOT_FOUND"
              : "PRECONDITION_FAILED",
        message: result.message,
      });
    }

    return { plan: result.plan };
  });
