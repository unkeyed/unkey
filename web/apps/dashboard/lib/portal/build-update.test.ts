import type { Portal } from "@unkey/api/models/components";
import { describe, expect, it } from "vitest";
import { type PortalFormValues, buildPortalUpdate, portalFormValues } from "./build-update";

const portal = (overrides: Partial<Portal> = {}): Portal => ({
  id: "pc_1234",
  slug: "acme",
  enabled: true,
  mapping: { type: "keyspace", id: "ks_1234" },
  createdAt: 1_700_000_000_000,
  ...overrides,
});

const branded = portal({
  branding: { logoUrl: "https://acme.com/logo.png", primaryColor: "#ff0000" },
});

const edit = (original: Portal, overrides: Partial<PortalFormValues>): PortalFormValues => ({
  ...portalFormValues(original),
  ...overrides,
});

describe("portalFormValues", () => {
  it("normalizes absent branding to empty strings", () => {
    expect(portalFormValues(portal())).toEqual({
      slug: "acme",
      enabled: true,
      logoUrl: "",
      primaryColor: "",
    });
  });
});

describe("buildPortalUpdate", () => {
  it("returns null when nothing changed", () => {
    expect(buildPortalUpdate(branded, portalFormValues(branded))).toBeNull();
  });

  it("sends only the slug when only the slug changed", () => {
    expect(buildPortalUpdate(branded, edit(branded, { slug: "acme-two" }))).toEqual({
      portal: "pc_1234",
      slug: "acme-two",
    });
  });

  it("clears a set logoUrl with null", () => {
    expect(buildPortalUpdate(branded, edit(branded, { logoUrl: "" }))).toEqual({
      portal: "pc_1234",
      logoUrl: null,
    });
  });

  it("clears a set primaryColor with null", () => {
    expect(buildPortalUpdate(branded, edit(branded, { primaryColor: "" }))).toEqual({
      portal: "pc_1234",
      primaryColor: null,
    });
  });

  it("omits a field that was already absent and stays empty", () => {
    const bare = portal();
    expect(buildPortalUpdate(bare, edit(bare, { slug: "acme-two" }))).toEqual({
      portal: "pc_1234",
      slug: "acme-two",
    });
  });

  it("sends enabled when it is toggled", () => {
    expect(buildPortalUpdate(branded, edit(branded, { enabled: false }))).toEqual({
      portal: "pc_1234",
      enabled: false,
    });
  });

  it("sends branding and slug together in one body", () => {
    const bare = portal();
    expect(
      buildPortalUpdate(
        bare,
        edit(bare, {
          slug: "acme-two",
          logoUrl: "https://acme.com/logo.png",
          primaryColor: "#00ff00",
        }),
      ),
    ).toEqual({
      portal: "pc_1234",
      slug: "acme-two",
      logoUrl: "https://acme.com/logo.png",
      primaryColor: "#00ff00",
    });
  });
});
