import type { DeployPlanOption } from "@/lib/trpc/routers/stripe/getDeployPlans";
import { describe, expect, it } from "vitest";
import { currentPlanState, isComputeUpgrade, planCardStates } from "./plan-card-state";

const option = (
  plan: DeployPlanOption["plan"],
  name: string,
  amount: number | null,
): DeployPlanOption => ({
  plan,
  name,
  amount,
  currency: "usd",
  interval: "month",
  description: null,
});

const OPTIONS = [
  option("starter", "Starter", 500),
  option("pro", "Pro", 2500),
  option("business", "Business", 5000),
];

describe("planCardStates", () => {
  it("shows every plan with loading prices before Stripe answers", () => {
    expect(
      planCardStates({ options: undefined, current: { status: "none" }, usageCents: null }).map(
        (card) => [card.name, card.price.type, card.action],
      ),
    ).toEqual([
      ["Starter", "loading", { type: "loading", label: "Choose Starter" }],
      ["Pro", "loading", { type: "loading", label: "Choose Pro" }],
      ["Business", "loading", { type: "loading", label: "Choose Business" }],
    ]);
  });

  it("offers every plan to a workspace without one", () => {
    const cards = planCardStates({
      options: OPTIONS,
      current: { status: "none" },
      usageCents: null,
    });
    expect(cards.map((card) => card.action.label)).toEqual([
      "Choose Starter",
      "Choose Pro",
      "Choose Business",
    ]);
    expect(cards[1]?.price).toEqual({ type: "amount", cents: 2500, interval: "month" });
  });

  it("marks the current plan and names upgrades and downgrades", () => {
    const cards = planCardStates({
      options: OPTIONS,
      current: { status: "plan", plan: "pro" },
      usageCents: null,
    });
    expect(cards.map((card) => [card.action.type, card.action.label])).toEqual([
      ["select", "Downgrade to Starter"],
      ["current", "Current plan"],
      ["select", "Upgrade to Business"],
    ]);
  });

  it("shows contact pricing when a plan has no fee", () => {
    const cards = planCardStates({
      options: [
        option("starter", "Starter", 500),
        option("pro", "Pro", 2500),
        option("business", "Business", null),
      ],
      current: { status: "none" },
      usageCents: null,
    });
    expect(cards[2]?.price).toEqual({ type: "contact" });
  });

  it("uses a neutral label when either price is unknown", () => {
    const cards = planCardStates({
      options: [
        option("starter", "Starter", 500),
        option("pro", "Pro", 2500),
        option("business", "Business", null),
      ],
      current: { status: "plan", plan: "business" },
      usageCents: null,
    });
    expect(cards.map((card) => card.action.label)).toEqual([
      "Switch to Starter",
      "Switch to Pro",
      "Current plan",
    ]);
  });

  it("warns when this period's usage exceeds a smaller plan's credits", () => {
    const cards = planCardStates({
      options: OPTIONS,
      current: { status: "plan", plan: "business" },
      usageCents: 3000,
    });
    expect(cards.map((card) => card.warning === null)).toEqual([false, false, true]);
    expect(cards[1]?.warning).toBe(
      "Your usage this period ($30.00) already exceeds the $25 of monthly credits Pro includes. This period keeps your current credits; from next period, usage at this level is billed as overage.",
    );
    expect(
      planCardStates({ options: OPTIONS, current: { status: "none" }, usageCents: 3000 }).every(
        (card) => card.warning === null,
      ),
    ).toBe(true);
  });

  it("does not warn on upgrades, which add credits", () => {
    const cards = planCardStates({
      options: OPTIONS,
      current: { status: "plan", plan: "starter" },
      usageCents: 3000,
    });
    expect(cards.map((card) => card.warning)).toEqual([null, null, null]);
  });

  it("holds every action until the current plan is known", () => {
    const cards = planCardStates({
      options: OPTIONS,
      current: { status: "loading" },
      usageCents: null,
    });
    expect(cards.map((card) => card.action)).toEqual([
      { type: "loading", label: "Choose Starter" },
      { type: "loading", label: "Choose Pro" },
      { type: "loading", label: "Choose Business" },
    ]);
  });

  it("blocks every action when the current plan could not be read", () => {
    const cards = planCardStates({
      options: OPTIONS,
      current: { status: "error" },
      usageCents: null,
    });
    expect(cards.map((card) => card.action)).toEqual([
      { type: "unavailable", label: "Choose Starter" },
      { type: "unavailable", label: "Choose Pro" },
      { type: "unavailable", label: "Choose Business" },
    ]);
  });
});

describe("currentPlanState", () => {
  it("prefers an error over a stale loading flag", () => {
    expect(currentPlanState({ isError: true, isLoading: true, plan: null })).toEqual({
      status: "error",
    });
  });

  it("stays loading until the query answers", () => {
    expect(currentPlanState({ isError: false, isLoading: true, plan: null })).toEqual({
      status: "loading",
    });
  });

  it("reads a plan or its absence", () => {
    expect(currentPlanState({ isError: false, isLoading: false, plan: "pro" })).toEqual({
      status: "plan",
      plan: "pro",
    });
    expect(currentPlanState({ isError: false, isLoading: false, plan: null })).toEqual({
      status: "none",
    });
  });
});

describe("isComputeUpgrade", () => {
  it("orders plans starter, pro, business", () => {
    expect(isComputeUpgrade("starter", "pro")).toBe(true);
    expect(isComputeUpgrade("pro", "business")).toBe(true);
    expect(isComputeUpgrade("business", "starter")).toBe(false);
    expect(isComputeUpgrade("pro", "pro")).toBe(false);
  });
});
