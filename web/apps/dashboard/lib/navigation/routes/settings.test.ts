import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";

describe("settings-scoped paths", () => {
  it("builds the settings leaf paths", () => {
    const scope = { workspaceSlug: ws };
    expect(routes.settings.general(scope)).toBe("/acme/settings/general");
    expect(routes.settings.team(scope)).toBe("/acme/settings/team");
    expect(routes.settings.rootKeys(scope)).toBe("/acme/settings/root-keys");
    expect(routes.settings.billing(scope)).toBe("/acme/settings/billing");
  });

  it("builds the stripe redirect paths", () => {
    const scope = { workspaceSlug: ws };
    expect(routes.settings.stripe.portal(scope)).toBe("/acme/stripe/portal");
    expect(routes.settings.stripe.checkout(scope)).toBe("/acme/stripe/checkout");
  });
});
