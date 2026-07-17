import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import {
  deployBillingConfig,
  deploySubscriptionItems,
  findDeployItems,
  findPlanFeeItem,
} from "@/lib/stripe/deployBilling";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import { isDeadSubscription } from "@/lib/stripe/subscriptionUtils";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";
import { assertSubscriptionAttachable } from "./subscriptionGuards";

/**
 * Subscribes the workspace to an Unkey Deploy plan: attaches the plan-fee price
 * for the chosen tier plus the shared metered prices to the workspace's Stripe
 * subscription, creating the subscription first if the workspace is on the free
 * tier with no subscription yet.
 *
 * Writes workspaces.deploy_plan optimistically so the UI reflects the new plan
 * immediately. Stripe stays source of truth: customer.subscription.* reconciles
 * deploy_plan on webhook retry.
 */
export const subscribeDeploy = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(
    z.object({
      plan: z.enum(DEPLOY_PLANS),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const config = await deployBillingConfig();
    if (!config) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Compute billing is not configured.",
      });
    }

    if (!ctx.workspace.stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Add a payment method before subscribing to a Compute plan.",
      });
    }

    if (ctx.workspace.deployPlan) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace already has a Compute plan. Use change instead.",
      });
    }

    const stripe = getStripeClient();
    const items = deploySubscriptionItems(config, input.plan);

    // Claim the Compute plan slot before touching Stripe. Two admins racing with
    // different plans both pass the deployPlan guard above (it reads a possibly
    // stale request snapshot) and Stripe only rejects duplicate SAME-price items,
    // so without this both plan-fee items would attach and both bill. The
    // compare-and-set on plan IS NULL serializes them: exactly one write matches a
    // row, the loser gets zero and bails before any Stripe charge. This also
    // writes the plan optimistically (the transaction below re-commits the same
    // value), so the UI reflects it immediately.
    const claim = await db
      .update(schema.workspaceBilling)
      .set({ plan: input.plan })
      .where(
        and(
          eq(schema.workspaceBilling.workspaceId, ctx.workspace.id),
          isNull(schema.workspaceBilling.plan),
        ),
      );
    if (claim[0].affectedRows !== 1) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace already has a Compute plan. Use change instead.",
      });
    }

    // Release the claim if anything below fails, so a genuine retry can proceed.
    // Conditioned on the plan we set, so a webhook that reconciled a different
    // value in the meantime is never clobbered. Best-effort: on the create path a
    // charged-but-unrecorded subscription is recovered by the workspace-scoped
    // idempotency key on retry, and the customer.subscription.* webhook reconciles
    // the plan regardless.
    const releaseClaim = async () => {
      await db
        .update(schema.workspaceBilling)
        .set({ plan: null })
        .where(
          and(
            eq(schema.workspaceBilling.workspaceId, ctx.workspace.id),
            eq(schema.workspaceBilling.plan, input.plan),
          ),
        );
    };

    try {
      // Columns to write once the Stripe mutation below succeeds, set in whichever
      // branch runs. Written optimistically so the UI reflects the new plan now;
      // the customer.subscription.* webhook reconciles them (a no-op, since it
      // derives the same value from the subscription we just mutated).
      let workspaceUpdate: { plan: string; stripeSubscriptionId?: string };

      // The recorded subscription can be a corpse (cancelDeploy cancels a
      // Compute-only subscription outright, and the deleted-webhook that clears
      // the column may not have landed yet). A dead one still carries its old
      // Compute items, so without this check a mid-month resubscribe dies on the
      // "already has Compute items" guard below. Treat it as absent and create a
      // fresh subscription instead.
      let existingSub: Stripe.Subscription | null = null;
      if (ctx.workspace.stripeSubscriptionId) {
        const recorded = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);
        if (!isDeadSubscription(recorded)) {
          existingSub = recorded;
        }
      }

      if (existingSub) {
        // Existing subscription (e.g. a paid API plan): append the Deploy items.
        // Items not listed here are left untouched, so API items are preserved.
        const sub = existingSub;

        const planFeeItem = findPlanFeeItem(config, sub.items.data);

        // A plan-fee item on a subscription that is NOT cancelling means the
        // workspace is actually subscribed; the deployPlan guard above should have
        // caught it, so it is a state mismatch, not something to double-attach
        // onto. A plan-fee item on a subscription set to cancel at period end is
        // the leftover of a Deploy-only cancel this period, which re-subscribing
        // resumes rather than rejecting.
        if (planFeeItem && !sub.cancel_at_period_end) {
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: "Workspace already has a Compute plan fee on its subscription.",
          });
        }

        // A genuine resume keeps the plan fee already paid this period, so it must
        // not re-invoice (proration "none" below). That only holds when the
        // resumed tier is the SAME one that was cancelled: repointing the fee at a
        // different tier under "none" would grant the new tier for the old tier's
        // payment, dodging the upgrade proration changeDeployPlan charges. Route a
        // tier change through changeDeployPlan (resume first, then switch) so the
        // proration and credit-grant rules apply exactly once, in one place.
        if (planFeeItem && sub.cancel_at_period_end && planFeeItem.plan !== input.plan) {
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: `Your cancelled Compute plan is ${planFeeItem.plan}. Resume it by subscribing to ${planFeeItem.plan} again, then switch to ${input.plan}.`,
          });
        }

        // Build the item set to attach without stranding a workspace whose cancel
        // left items behind. A Deploy-only cancel sets cancel_at_period_end and
        // keeps every item, so point the existing plan-fee item at the chosen
        // tier. A mixed cancel drops the plan fee but keeps the metered items, so
        // add the plan fee back. Either way, only add metered prices not already
        // present so they are never duplicated.
        const existingPriceIds = new Set(
          findDeployItems(config, sub.items.data).map((item) => item.priceId),
        );
        const itemsToAttach: Stripe.SubscriptionUpdateParams.Item[] = [
          planFeeItem
            ? { id: planFeeItem.id, price: config.planFeePriceIds[input.plan] }
            : { price: config.planFeePriceIds[input.plan] },
          ...config.meteredPriceIds
            .filter((price) => !existingPriceIds.has(price))
            .map((price) => ({ price })),
        ];

        // Resuming a subscription that was cancelling this period just clears the
        // pending cancellation. It requires a leftover Compute plan-fee item:
        // without one (e.g. a cancelling API-only subscription) there is nothing
        // Compute to resume, so falling through to assertSubscriptionAttachable
        // rejects the cancelling subscription and tells the user to resume it
        // first, instead of un-cancelling an API plan they killed and attaching
        // Compute uninvoiced. A fresh attach onto another subscription must first
        // be attachable, so we don't attach Deploy items to something that ends at
        // the period roll.
        const resuming = sub.cancel_at_period_end && planFeeItem !== undefined;
        if (!resuming) {
          assertSubscriptionAttachable(sub);
        }

        try {
          await stripe.subscriptions.update(sub.id, {
            cancel_at_period_end: false,
            items: itemsToAttach,
            // Resuming keeps the plan fee already paid this period (cancel never
            // refunded it), so it must not invoice again; a fresh attach bills it.
            proration_behavior: resuming ? "none" : "always_invoice",
            payment_behavior: "error_if_incomplete",
          });
        } catch (err) {
          throw toBillingError(err);
        }

        workspaceUpdate = { plan: input.plan };
      } else {
        // Free tier: create a subscription whose initial items are the Deploy set.
        // error_if_incomplete keeps us off a half-paid state if the card declines.
        //
        // subscriptions.create only consults the customer's DEFAULT payment
        // method, and a card that arrived via subscription-mode Checkout is
        // attached but recorded as the (now possibly cancelled) subscription's
        // default, not the customer's — so a cancel-then-resubscribe would die
        // with "no attached payment source". Resolve one explicitly: use the
        // customer default when set, else the most recently attached method.
        let defaultPaymentMethod: string | undefined;
        const customer = await stripe.customers.retrieve(ctx.workspace.stripeCustomerId);
        const hasCustomerDefault =
          !customer.deleted &&
          Boolean(customer.invoice_settings?.default_payment_method || customer.default_source);
        if (!hasCustomerDefault) {
          const attached = await stripe.customers.listPaymentMethods(
            ctx.workspace.stripeCustomerId,
            {
              limit: 1,
            },
          );
          defaultPaymentMethod = attached.data[0]?.id;
          if (!defaultPaymentMethod) {
            // BAD_REQUEST on purpose: the projects-landing subscriber treats it
            // as "card problem" and routes the user to Stripe checkout to add one.
            throw new TRPCError({
              code: "BAD_REQUEST",
              message: "No payment method on file. Add one to subscribe.",
            });
          }
        }

        const createParams: Stripe.SubscriptionCreateParams = {
          customer: ctx.workspace.stripeCustomerId,
          items,
          ...(defaultPaymentMethod ? { default_payment_method: defaultPaymentMethod } : {}),
          billing_cycle_anchor_config: { day_of_month: 1, hour: 0, minute: 0, second: 0 },
          // Pin classic billing mode (clover defaults new subscriptions to
          // "flexible", which itemizes prorations differently); see the same
          // pin in createSubscription.
          billing_mode: { type: "classic" },
          proration_behavior: "always_invoice",
          payment_behavior: "error_if_incomplete",
        };

        let sub: Stripe.Subscription;
        try {
          // Deterministic idempotency key: if Stripe created (and charged) the
          // subscription but the workspace write below failed, a retry replays
          // the SAME subscription instead of charging a second one. Unlike the
          // checkout-session path there is no webhook backstop for this create —
          // customer.subscription.created resolves the workspace by
          // stripeSubscriptionId, which was never written.
          //
          // The key is scoped to the workspace, NOT the plan: a retry that picks
          // a different plan (or two admins racing on different plans, though the
          // compare-and-set above already blocks that) must still replay the one
          // subscription rather than mint a second charged one. The first subscribe
          // wins the plan; a later tier change goes through changeDeployPlan.
          //
          // Stripe's idempotency layer replays the CREATION-TIME response, so
          // after a full subscribe→cancel cycle inside the key window a replay
          // hands back the canceled subscription still claiming to be active.
          // Re-retrieve the live status and, on a corpse, chain a fresh key off
          // its id and mint a new subscription (a retry of THIS request replays
          // that same fresh subscription) — mirroring stripe/checkout/page.tsx.
          let idempotencyKey = `deploy-subscribe:${ctx.workspace.id}`;
          sub = await stripe.subscriptions.create(createParams, { idempotencyKey });
          for (let attempt = 0; attempt < 3; attempt++) {
            const live = await stripe.subscriptions.retrieve(sub.id);
            if (!isDeadSubscription(live)) {
              sub = live;
              break;
            }
            idempotencyKey = `${idempotencyKey}:${live.id}`;
            sub = await stripe.subscriptions.create(createParams, { idempotencyKey });
          }
        } catch (err) {
          throw toBillingError(err);
        }

        if (sub.status !== "active" && sub.status !== "trialing") {
          try {
            await stripe.subscriptions.cancel(sub.id);
          } catch (cancelErr) {
            console.error(
              `Failed to cancel non-active subscription ${sub.id} after creation:`,
              cancelErr,
            );
          }
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: `Subscription was created but is not active (ID: ${sub.id}). Please contact support.`,
          });
        }

        // Link the new subscription so the customer.subscription.* webhook can
        // resolve this workspace.
        workspaceUpdate = { stripeSubscriptionId: sub.id, plan: input.plan };
      }

      // One transaction so the plan write and its audit log commit together; a
      // failure in either rolls back the other.
      await db.transaction(async (tx) => {
        await tx
          .update(schema.workspaceBilling)
          .set(workspaceUpdate)
          .where(eq(schema.workspaceBilling.workspaceId, ctx.workspace.id));
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.update",
          description: `Subscribed to Compute ${input.plan} plan.`,
          resources: [],
          context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
        });
      });
    } catch (err) {
      await releaseClaim().catch((releaseErr) => {
        console.error("Failed to release Compute plan claim after subscribe error", {
          workspaceId: ctx.workspace.id,
          error: releaseErr instanceof Error ? releaseErr.message : releaseErr,
        });
      });
      throw err;
    }
  });

/**
 * Surfaces Stripe payment errors as actionable TRPCErrors; rethrows anything
 * else so genuine bugs are not masked as a card problem.
 */
function toBillingError(err: unknown): TRPCError {
  if (err instanceof Stripe.errors.StripeCardError || err instanceof Stripe.errors.StripeError) {
    return new TRPCError({
      code: "BAD_REQUEST",
      message:
        err.message ||
        "Payment could not be completed. Please update your payment method and try again.",
    });
  }
  throw err;
}
