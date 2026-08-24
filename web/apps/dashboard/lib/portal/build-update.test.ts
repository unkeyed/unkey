import type { Portal } from "@unkey/api/models/components";
import { describe, expect, it } from "vitest";
import {
  type PortalDirtyFields,
  type PortalFormValues,
  buildPortalUpdate,
  portalFormValues,
} from "./build-update";

const portal = (overrides: Partial<Portal> = {}): Portal => ({
  id: "pc_1234",
  slug: "acme",
  displayName: "Acme",
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

/** The dirty record react-hook-form would produce for `overrides`. */
const dirty = (...fields: (keyof PortalFormValues)[]): PortalDirtyFields =>
  Object.fromEntries(fields.map((field) => [field, true]));

describe("portalFormValues", () => {
  it("normalizes absent branding to empty strings", () => {
    expect(portalFormValues(portal())).toEqual({
      slug: "acme",
      displayName: "Acme",
      enabled: true,
      logoUrl: "",
      primaryColor: "",
    });
  });
});

describe("buildPortalUpdate", () => {
  it("returns null when no field is dirty", () => {
    expect(buildPortalUpdate("pc_1234", portalFormValues(branded), {})).toBeNull();
  });

  it("returns null when every dirty flag is false", () => {
    expect(
      buildPortalUpdate("pc_1234", portalFormValues(branded), {
        slug: false,
        logoUrl: false,
        primaryColor: false,
      }),
    ).toBeNull();
  });

  it("sends only the slug when only the slug is dirty", () => {
    expect(
      buildPortalUpdate("pc_1234", edit(branded, { slug: "acme-two" }), dirty("slug")),
    ).toEqual({
      portal: "pc_1234",
      slug: "acme-two",
    });
  });

  it("clears an emptied logoUrl with null", () => {
    expect(buildPortalUpdate("pc_1234", edit(branded, { logoUrl: "" }), dirty("logoUrl"))).toEqual({
      portal: "pc_1234",
      logoUrl: null,
    });
  });

  it("clears an emptied primaryColor with null", () => {
    expect(
      buildPortalUpdate("pc_1234", edit(branded, { primaryColor: "" }), dirty("primaryColor")),
    ).toEqual({
      portal: "pc_1234",
      primaryColor: null,
    });
  });

  it("omits a branding field the operator never touched", () => {
    // The values carry a logo the form never edited, e.g. a refetched portal.
    expect(
      buildPortalUpdate("pc_1234", edit(branded, { slug: "acme-two" }), dirty("slug")),
    ).not.toHaveProperty("logoUrl");
  });

  it("sends enabled when it is dirty", () => {
    expect(
      buildPortalUpdate("pc_1234", edit(branded, { enabled: false }), dirty("enabled")),
    ).toEqual({
      portal: "pc_1234",
      enabled: false,
    });
  });

  it("sends branding and slug together in one body", () => {
    const bare = portal();
    expect(
      buildPortalUpdate(
        "pc_1234",
        edit(bare, {
          slug: "acme-two",
          logoUrl: "https://acme.com/logo.png",
          primaryColor: "#00ff00",
        }),
        dirty("slug", "logoUrl", "primaryColor"),
      ),
    ).toEqual({
      portal: "pc_1234",
      slug: "acme-two",
      logoUrl: "https://acme.com/logo.png",
      primaryColor: "#00ff00",
    });
  });
});
