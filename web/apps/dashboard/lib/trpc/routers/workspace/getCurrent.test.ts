import { initTRPC } from "@trpc/server";
import { beforeEach, expect, it, vi } from "vitest";
import { getCurrentWorkspace } from "./getCurrent";

const mocks = vi.hoisted(() => ({ where: vi.fn() }));

vi.mock("@/lib/db", () => ({
  db: {
    select: () => {
      const query = { from: () => query, leftJoin: () => query, where: mocks.where };
      return query;
    },
  },
}));

vi.mock("../../trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  return { protectedProcedure: initTRPC.create().procedure };
});

const router = initTRPC
  .context<{
    workspace?: { id: string; flags: Record<string, boolean> };
    tenant: { id: string } | null;
  }>()
  .create()
  .router({ getCurrent: getCurrentWorkspace });

beforeEach(() => {
  vi.resetAllMocks();
});

it("returns flags already loaded with the workspace", async () => {
  const workspace = { id: "ws_owned", flags: { preview: false, routing: true } };
  const result = await router
    .createCaller({ workspace, tenant: { id: "org_session" } })
    .getCurrent();
  expect(result).toEqual(workspace);
});

it("returns flags from the fallback workspace query", async () => {
  mocks.where.mockResolvedValue([
    {
      workspace: { id: "ws_fallback" },
      limits: null,
      billing: null,
      subscription: null,
      flag: { slug: "preview", defaultValue: false },
      override: { value: true },
    },
  ]);
  const result = await router.createCaller({ tenant: { id: "org_session" } }).getCurrent();
  expect(result).toMatchObject({ id: "ws_fallback", flags: { preview: true } });
});

it("preserves billing and flag values without duplicating subscriptions across joined rows", async () => {
  const billing = {
    tier: "Pro",
    stripeCustomerId: "cus_owned",
    plan: "hobby",
    planOverride: "pro",
    spendBudgetCents: 2500,
    spendBudgetStop: true,
    spendSuspended: false,
  };
  const subscriptions = [
    { product: "api", stripeSubscriptionId: "sub_api" },
    { product: "compute", stripeSubscriptionId: "sub_compute" },
  ];
  mocks.where.mockResolvedValue(
    subscriptions.flatMap((subscription) =>
      [
        { flag: { slug: "disabled", defaultValue: true }, override: { value: false } },
        { flag: { slug: "enabled", defaultValue: false }, override: { value: true } },
        { flag: { slug: "inherited", defaultValue: true }, override: null },
        { flag: { slug: "private", defaultValue: false }, override: null },
      ].map((flag) => ({
        workspace: { id: "ws_owned" },
        limits: { teamEnabled: true },
        billing,
        subscription,
        ...flag,
      })),
    ),
  );
  const result = await router.createCaller({ tenant: { id: "org_session" } }).getCurrent();
  expect(result).toEqual({
    id: "ws_owned",
    limits: { teamEnabled: true },
    billing,
    billingSubscriptions: subscriptions,
    flags: { disabled: false, enabled: true, inherited: true, private: false },
    tier: "Pro",
    stripeCustomerId: "cus_owned",
    stripeSubscriptionId: "sub_api",
    stripeDeploySubscriptionId: "sub_compute",
    deployPlan: "hobby",
    deployPlanOverride: "pro",
    deploySpendBudgetCents: 2500,
    deploySpendBudgetStop: true,
    deploySpendSuspended: false,
  });
});

it("keeps a workspace with no flags, billing, subscriptions, or limits", async () => {
  mocks.where.mockResolvedValue([
    {
      workspace: { id: "ws_empty" },
      limits: null,
      billing: null,
      subscription: null,
      flag: null,
      override: null,
    },
  ]);
  const result = await router.createCaller({ tenant: { id: "org_session" } }).getCurrent();
  expect(result).toEqual({
    id: "ws_empty",
    limits: null,
    billing: null,
    billingSubscriptions: [],
    flags: {},
    tier: "Free",
    stripeCustomerId: null,
    stripeSubscriptionId: null,
    stripeDeploySubscriptionId: null,
    deployPlan: null,
    deployPlanOverride: null,
    deploySpendBudgetCents: null,
    deploySpendBudgetStop: false,
    deploySpendSuspended: false,
  });
});
