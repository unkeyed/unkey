import type { Environment } from "@/lib/collections/deploy/environments";
import { describe, expect, it } from "vitest";
import {
  environmentRedirectPath,
  resolveEnvironmentRoute,
  withEnvironmentSlug,
} from "./environment-route";

const production: Environment = {
  id: "env_prod",
  projectId: "proj_1",
  appId: "app_1",
  slug: "prod",
  kind: "production",
};
const preview: Environment = {
  id: "env_preview",
  projectId: "proj_1",
  appId: "app_1",
  slug: "preview",
  kind: "preview",
};
const staging: Environment = { ...preview, id: "env_staging", slug: "staging" };

const base = "/acme/projects/proj_1/apps/app_1";

describe("resolveEnvironmentRoute", () => {
  it("resolves a known slug", () => {
    expect(resolveEnvironmentRoute("staging", [production, preview, staging])).toEqual({
      kind: "ok",
      environment: staging,
    });
  });

  it("treats page names as legacy urls and points them at the production slug", () => {
    expect(resolveEnvironmentRoute("settings", [preview, production])).toEqual({
      kind: "legacy",
      redirectSlug: "prod",
    });
  });

  it("falls back to the first environment when there is no production one", () => {
    expect(resolveEnvironmentRoute("deployments", [staging, preview])).toEqual({
      kind: "legacy",
      redirectSlug: "staging",
    });
  });

  it("redirects an unknown slug to production", () => {
    expect(resolveEnvironmentRoute("prodction", [production, preview])).toEqual({
      kind: "unknown",
      redirectSlug: "prod",
    });
  });

  it("is notFound when the app has no environments", () => {
    expect(resolveEnvironmentRoute("settings", [])).toEqual({ kind: "notFound" });
    expect(resolveEnvironmentRoute("preview", [])).toEqual({ kind: "notFound" });
  });
});

describe("withEnvironmentSlug", () => {
  it("keeps the sub-page when switching environments", () => {
    expect(withEnvironmentSlug(`${base}/prod/settings`, "app_1", "preview")).toBe(
      `${base}/preview/settings`,
    );
    expect(withEnvironmentSlug(`${base}/prod/deployments/dep_1`, "app_1", "preview")).toBe(
      `${base}/preview/deployments/dep_1`,
    );
  });

  it("lands on the overview when the path ends at the environment", () => {
    expect(withEnvironmentSlug(`${base}/prod`, "app_1", "preview")).toBe(
      `${base}/preview/overview`,
    );
  });
});

describe("environmentRedirectPath", () => {
  it("inserts the slug in front of a legacy page path", () => {
    expect(
      environmentRedirectPath(`${base}/deployments/dep_1`, "app_1", {
        kind: "legacy",
        redirectSlug: "prod",
      }),
    ).toBe(`${base}/prod/deployments/dep_1`);
    expect(
      environmentRedirectPath(`${base}/settings`, "app_1", {
        kind: "legacy",
        redirectSlug: "prod",
      }),
    ).toBe(`${base}/prod/settings`);
  });

  it("replaces an unknown slug and keeps the rest of the path", () => {
    expect(
      environmentRedirectPath(`${base}/prodction/settings`, "app_1", {
        kind: "unknown",
        redirectSlug: "prod",
      }),
    ).toBe(`${base}/prod/settings`);
    expect(
      environmentRedirectPath(`${base}/prodction`, "app_1", {
        kind: "unknown",
        redirectSlug: "prod",
      }),
    ).toBe(`${base}/prod/overview`);
  });
});
