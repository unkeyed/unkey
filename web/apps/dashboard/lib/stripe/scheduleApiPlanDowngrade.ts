import type Stripe from "stripe";

const API_PLAN_DOWNGRADE = "api_plan_downgrade";

type ScheduleApiPlanDowngradeInput = {
  subscriptionId: string;
  schedule: Stripe.Subscription["schedule"];
  currentPriceId: string;
  newPriceId: string;
};

export class SubscriptionScheduleConflictError extends Error {}

function resourceId(resource: string | { id: string }): string {
  return typeof resource === "string" ? resource : resource.id;
}

function phaseItems(
  items: Stripe.SubscriptionSchedule.Phase.Item[],
  priceChange?: { from: string; to: string },
): Stripe.SubscriptionScheduleUpdateParams.Phase.Item[] {
  return items.map((item) => ({
    price:
      priceChange && resourceId(item.price) === priceChange.from
        ? priceChange.to
        : resourceId(item.price),
    ...(item.quantity === undefined ? {} : { quantity: item.quantity }),
  }));
}

async function retrieveSchedule(
  stripe: Stripe,
  schedule: Exclude<Stripe.Subscription["schedule"], null>,
): Promise<Stripe.SubscriptionSchedule> {
  return typeof schedule === "string" ? stripe.subscriptionSchedules.retrieve(schedule) : schedule;
}

function assertManagedSchedule(schedule: Stripe.SubscriptionSchedule): void {
  if (schedule.metadata?.unkey_change !== API_PLAN_DOWNGRADE) {
    throw new SubscriptionScheduleConflictError(
      "This subscription already has a different scheduled change.",
    );
  }
}

/** Keeps the current price active and changes it at the next billing boundary. */
export async function scheduleApiPlanDowngrade(
  stripe: Stripe,
  input: ScheduleApiPlanDowngradeInput,
): Promise<{ effectiveAt: number }> {
  const created = input.schedule === null;
  const schedule = input.schedule
    ? await retrieveSchedule(stripe, input.schedule)
    : await stripe.subscriptionSchedules.create({ from_subscription: input.subscriptionId });

  if (input.schedule) {
    assertManagedSchedule(schedule);
  }

  try {
    const currentPhase = schedule.current_phase
      ? schedule.phases.find(
          (phase) =>
            phase.start_date === schedule.current_phase?.start_date &&
            phase.end_date === schedule.current_phase.end_date,
        )
      : undefined;
    if (!currentPhase) {
      throw new Error("Stripe did not return the current subscription schedule phase.");
    }

    const hasCurrentPrice = currentPhase.items.some(
      (item) => resourceId(item.price) === input.currentPriceId,
    );
    if (!hasCurrentPrice) {
      throw new Error("The current API price was not found in the subscription schedule.");
    }

    const currentItems = phaseItems(currentPhase.items);
    const futureItems = phaseItems(currentPhase.items, {
      from: input.currentPriceId,
      to: input.newPriceId,
    });

    await stripe.subscriptionSchedules.update(schedule.id, {
      end_behavior: "release",
      metadata: {
        ...(schedule.metadata ?? {}),
        unkey_change: API_PLAN_DOWNGRADE,
      },
      // Updating the unchanged current phase must never create a credit/refund.
      proration_behavior: "none",
      phases: [
        {
          items: currentItems,
          start_date: currentPhase.start_date,
          end_date: currentPhase.end_date,
          proration_behavior: "none",
        },
        {
          items: futureItems,
          duration: { interval: "month", interval_count: 1 },
          // The lower price starts on the renewal boundary, without a proration.
          proration_behavior: "none",
        },
      ],
    });

    return { effectiveAt: currentPhase.end_date * 1000 };
  } catch (error) {
    // from_subscription and phase configuration require separate Stripe calls.
    // Undo the first call if the second fails so the customer can retry.
    if (created) {
      await stripe.subscriptionSchedules.release(schedule.id);
    }
    throw error;
  }
}

/** Removes a pending API downgrade before an immediate plan change. */
export async function releaseScheduledApiPlanDowngrade(
  stripe: Stripe,
  scheduleReference: Exclude<Stripe.Subscription["schedule"], null>,
): Promise<void> {
  const schedule = await retrieveSchedule(stripe, scheduleReference);
  assertManagedSchedule(schedule);
  await stripe.subscriptionSchedules.release(schedule.id);
}
