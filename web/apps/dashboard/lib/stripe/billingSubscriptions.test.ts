import { describe, expect, it } from "vitest";
import { subscriptionIdFor, subscriptionIdsByProduct } from "./billingSubscriptions";

describe("subscriptionIdsByProduct", () => {
  it("maps each product's subscription id to its field", () => {
    expect(
      subscriptionIdsByProduct([
        { product: "api", stripeSubscriptionId: "sub_api" },
        { product: "compute", stripeSubscriptionId: "sub_deploy" },
      ]),
    ).toEqual({ stripeSubscriptionId: "sub_api", stripeDeploySubscriptionId: "sub_deploy" });
  });

  it("leaves a missing product null", () => {
    expect(
      subscriptionIdsByProduct([{ product: "compute", stripeSubscriptionId: "sub_deploy" }]),
    ).toEqual({ stripeSubscriptionId: null, stripeDeploySubscriptionId: "sub_deploy" });
  });

  it("returns both null for no subscriptions", () => {
    expect(subscriptionIdsByProduct([])).toEqual({
      stripeSubscriptionId: null,
      stripeDeploySubscriptionId: null,
    });
  });
});

describe("subscriptionIdFor", () => {
  const workspace = { stripeSubscriptionId: "sub_api", stripeDeploySubscriptionId: "sub_deploy" };

  it("returns the API subscription id for api", () => {
    expect(subscriptionIdFor(workspace, "api")).toBe("sub_api");
  });

  it("returns the deploy subscription id for compute", () => {
    expect(subscriptionIdFor(workspace, "compute")).toBe("sub_deploy");
  });

  it("returns null when the product has no subscription", () => {
    expect(
      subscriptionIdFor(
        { stripeSubscriptionId: null, stripeDeploySubscriptionId: "sub_deploy" },
        "api",
      ),
    ).toBeNull();
  });
});
