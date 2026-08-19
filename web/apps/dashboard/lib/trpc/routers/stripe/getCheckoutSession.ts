import { getStripeClient } from "@/lib/stripe";
import { expandableId, retrieveWorkspaceCheckoutSession } from "@/lib/trpc/routers/utils/stripe";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

const checkoutSessionSchema = z.object({
  id: z.string(),
  customer: z.string().nullable(),
  client_reference_id: z.string().nullable(),
  setup_intent: z.string().nullable(),
  // Populated for subscription-mode sessions (the Compute deploy path); null
  // for setup-mode sessions. `/success` branches on `mode`/`subscription` to
  // choose the link path vs. the legacy setup-intent path.
  subscription: z.string().nullable(),
  mode: z.string().nullable(),
  payment_status: z.string(),
  status: z.string(),
});

export const getCheckoutSession = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      sessionId: z.string().min(1, "Stripe checkout session ID is required"),
    }),
  )
  .output(checkoutSessionSchema)
  .query(async ({ ctx, input }) => {
    const stripe = getStripeClient();

    const session = await retrieveWorkspaceCheckoutSession({
      stripe,
      sessionId: input.sessionId,
      workspaceId: ctx.workspace.id,
      notFoundMessage: "Checkout session not found",
    });

    return {
      id: session.id,
      customer: expandableId(session.customer),
      client_reference_id: session.client_reference_id,
      setup_intent: expandableId(session.setup_intent),
      subscription: expandableId(session.subscription),
      mode: session.mode ?? null,
      payment_status: session.payment_status,
      status: session.status || "",
    };
  });
