import { initTRPC } from "@trpc/server";
import { beforeEach, expect, it, vi } from "vitest";
import { getCurrentWorkspace } from "./getCurrent";

const mocks = vi.hoisted(() => ({ findFirst: vi.fn() }));

const defaults = {
  "hello-world": false,
  "deploy-billing": false,
  "billing-ui-upgrades": false,
  "show-darksouls-success-banner": false,
  "projects-nav": false,
  "portal-management": false,
};

vi.mock("@/lib/db", () => ({
  db: { query: { workspaces: { findFirst: mocks.findFirst } } },
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

it("returns the fallback workspace without unregistered flags", async () => {
  mocks.findFirst.mockResolvedValue({
    id: "ws_fallback",
    limits: null,
    billing: null,
    billingSubscriptions: [],
    flags: { "projects-nav": 1, preview: 1 },
  });
  const result = await router.createCaller({ tenant: { id: "org_session" } }).getCurrent();
  expect(result).toMatchObject({ id: "ws_fallback", flags: { ...defaults, "projects-nav": true } });
});

it("preserves billing values from the workspace query", async () => {
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
  mocks.findFirst.mockResolvedValue({
    id: "ws_owned",
    limits: { teamEnabled: true },
    billing,
    billingSubscriptions: subscriptions,
    flags: {},
  });
  const result = await router.createCaller({ tenant: { id: "org_session" } }).getCurrent();
  expect(result).toEqual({
    id: "ws_owned",
    limits: { teamEnabled: true },
    billing,
    billingSubscriptions: subscriptions,
    flags: defaults,
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
  mocks.findFirst.mockResolvedValue({
    id: "ws_empty",
    limits: null,
    billing: null,
    billingSubscriptions: [],
    flags: {},
  });
  const result = await router.createCaller({ tenant: { id: "org_session" } }).getCurrent();
  expect(result).toEqual({
    id: "ws_empty",
    limits: null,
    billing: null,
    billingSubscriptions: [],
    flags: defaults,
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
