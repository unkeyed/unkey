import type Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";
import {
  SubscriptionScheduleConflictError,
  releaseScheduledApiPlanDowngrade,
  scheduleApiPlanDowngrade,
} from "./scheduleApiPlanDowngrade";

function schedule(
  overrides: Partial<Stripe.SubscriptionSchedule> = {},
): Stripe.SubscriptionSchedule {
  return {
    id: "sub_sched_1",
    current_phase: { start_date: 1_700_000_000, end_date: 1_702_678_400 },
    metadata: {},
    phases: [
      {
        start_date: 1_700_000_000,
        end_date: 1_702_678_400,
        items: [
          {
            price: "price_current",
            quantity: 1,
            metadata: {},
            discounts: [],
            tax_rates: [],
            billing_thresholds: null,
          },
          {
            price: "price_other",
            quantity: 2,
            metadata: {},
            discounts: [],
            tax_rates: [],
            billing_thresholds: null,
          },
        ],
        metadata: { workspace_id: "ws_1" },
        discounts: [],
        default_tax_rates: [],
        trial_end: 1_701_000_000,
      },
    ],
    ...overrides,
  } as unknown as Stripe.SubscriptionSchedule;
}

function stubStripe(createdSchedule = schedule(), retrievedSchedule = createdSchedule) {
  const create = vi.fn(async () => createdSchedule);
  const retrieve = vi.fn(async () => retrievedSchedule);
  const update = vi.fn(
    async (_id: string, _params: Stripe.SubscriptionScheduleUpdateParams) => retrievedSchedule,
  );
  const release = vi.fn(async () => retrievedSchedule);
  return {
    stripe: {
      subscriptionSchedules: { create, retrieve, update, release },
    } as unknown as Stripe,
    create,
    retrieve,
    update,
    release,
  };
}

describe("scheduleApiPlanDowngrade", () => {
  it("keeps the current price until renewal and disables all proration", async () => {
    const { stripe, create, update } = stubStripe();

    const result = await scheduleApiPlanDowngrade(stripe, {
      subscriptionId: "sub_1",
      schedule: null,
      currentPriceId: "price_current",
      newPriceId: "price_lower",
    });

    expect(result).toEqual({ effectiveAt: 1_702_678_400_000 });
    expect(create).toHaveBeenCalledWith({ from_subscription: "sub_1" });
    expect(update).toHaveBeenCalledWith("sub_sched_1", {
      end_behavior: "release",
      metadata: { unkey_change: "api_plan_downgrade" },
      proration_behavior: "none",
      phases: [
        {
          items: [
            {
              price: "price_current",
              quantity: 1,
            },
            {
              price: "price_other",
              quantity: 2,
            },
          ],
          start_date: 1_700_000_000,
          end_date: 1_702_678_400,
          proration_behavior: "none",
        },
        {
          items: [
            {
              price: "price_lower",
              quantity: 1,
            },
            {
              price: "price_other",
              quantity: 2,
            },
          ],
          duration: { interval: "month", interval_count: 1 },
          proration_behavior: "none",
        },
      ],
    });
  });

  it("replaces a downgrade previously scheduled by this flow", async () => {
    const managedSchedule = schedule({ metadata: { unkey_change: "api_plan_downgrade" } });
    const { stripe, create, retrieve, update } = stubStripe(schedule(), managedSchedule);

    await scheduleApiPlanDowngrade(stripe, {
      subscriptionId: "sub_1",
      schedule: "sub_sched_1",
      currentPriceId: "price_current",
      newPriceId: "price_different",
    });

    expect(create).not.toHaveBeenCalled();
    expect(retrieve).toHaveBeenCalledWith("sub_sched_1");
    expect(update).toHaveBeenCalledOnce();
    expect(update.mock.calls[0]?.[1].phases?.[1].items[0]?.price).toBe("price_different");
  });

  it("does not overwrite a schedule owned by another flow", async () => {
    const { stripe, update } = stubStripe(schedule(), schedule({ metadata: {} }));

    await expect(
      scheduleApiPlanDowngrade(stripe, {
        subscriptionId: "sub_1",
        schedule: "sub_sched_other",
        currentPriceId: "price_current",
        newPriceId: "price_lower",
      }),
    ).rejects.toBeInstanceOf(SubscriptionScheduleConflictError);
    expect(update).not.toHaveBeenCalled();
  });

  it("releases a newly created schedule when phase configuration fails", async () => {
    const { stripe, update, release } = stubStripe();
    const error = new Error("schedule update failed");
    update.mockRejectedValueOnce(error);

    await expect(
      scheduleApiPlanDowngrade(stripe, {
        subscriptionId: "sub_1",
        schedule: null,
        currentPriceId: "price_current",
        newPriceId: "price_lower",
      }),
    ).rejects.toBe(error);
    expect(release).toHaveBeenCalledWith("sub_sched_1");
  });
});

describe("releaseScheduledApiPlanDowngrade", () => {
  it("releases a downgrade owned by this flow", async () => {
    const managedSchedule = schedule({ metadata: { unkey_change: "api_plan_downgrade" } });
    const { stripe, release } = stubStripe(schedule(), managedSchedule);

    await releaseScheduledApiPlanDowngrade(stripe, "sub_sched_1");

    expect(release).toHaveBeenCalledWith("sub_sched_1");
  });
});
