import { describe, expect, it } from "vitest";
import {
  appNameFromImage,
  appNameFromRepo,
  findReusablePlaceholder,
  uniqueAppName,
} from "./app-name";

describe("appNameFromRepo", () => {
  it.each([
    ["acme/storefront", "storefront"],
    ["acme/Storefront_API", "storefront-api"],
    ["acme/web.gateway", "web-gateway"],
    ["acme/___", "app"],
    ["acme/ui", "ui-app"],
  ])("names %s as %s", (repo, name) => {
    expect(appNameFromRepo(repo)).toBe(name);
  });
});

describe("appNameFromImage", () => {
  it.each([
    ["ghcr.io/acme/api:1.2", "api"],
    ["nginx", "nginx"],
    ["localhost:5000/acme/worker:latest", "worker"],
    ["public.ecr.aws/acme/mcp-server@sha256:9f2c", "mcp-server"],
    ["quay.io/acme/api:v1@sha256:9f2c", "api"],
  ])("names %s as %s", (image, name) => {
    expect(appNameFromImage(image)).toBe(name);
  });
});

describe("uniqueAppName", () => {
  it("keeps a free name", () => {
    expect(uniqueAppName("storefront", new Set(["api"]))).toBe("storefront");
  });

  it("suffixes a taken name with the first free number", () => {
    expect(uniqueAppName("storefront", new Set(["storefront", "storefront-2"]))).toBe(
      "storefront-3",
    );
  });
});

describe("findReusablePlaceholder", () => {
  const placeholder = {
    id: "app_1",
    slug: "new-app-2",
    sourceType: "git" as const,
    repositoryFullName: null,
    currentDeploymentId: null,
    headlineDeployment: null,
  };

  it("reuses an unconnected, undeployed placeholder", () => {
    expect(findReusablePlaceholder([placeholder])).toBe("app_1");
  });

  it.each([
    ["a real name", { slug: "storefront" }],
    ["a connected repository", { repositoryFullName: "acme/storefront" }],
    ["a deployment", { currentDeploymentId: "d_1" }],
    ["an image source", { sourceType: "oci" as const }],
  ])("skips an app with %s", (_, patch) => {
    expect(findReusablePlaceholder([{ ...placeholder, ...patch }])).toBeNull();
  });
});
