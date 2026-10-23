import { describe, expect, it } from "vitest";
import {
  appNameError,
  appNameFromImage,
  appNameFromRepo,
  findReusablePlaceholder,
  hasGeneratedName,
  isPlaceholderApp,
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
    [
      "public.ecr.aws/acme/mcp-server@sha256:9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c",
      "mcp-server",
    ],
    [
      "quay.io/acme/api:v1@sha256:9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c9f2c",
      "api",
    ],
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

const placeholder = {
  id: "app_1",
  slug: "new-app-2",
  sourceType: "git" as const,
  repositoryFullName: null,
  currentDeploymentId: null,
  headlineDeployment: null,
};

describe("hasGeneratedName", () => {
  it("matches a name the flow made from the linked repository", () => {
    expect(hasGeneratedName({ slug: "storefront", repositoryFullName: "acme/storefront" })).toBe(
      true,
    );
    expect(hasGeneratedName({ slug: "storefront-2", repositoryFullName: "acme/storefront" })).toBe(
      true,
    );
  });

  it("leaves a chosen name or an unlinked app alone", () => {
    expect(hasGeneratedName({ slug: "my-shop", repositoryFullName: "acme/storefront" })).toBe(
      false,
    );
    expect(hasGeneratedName({ slug: "storefront", repositoryFullName: null })).toBe(false);
  });
});

describe("isPlaceholderApp", () => {
  it("marks the flow's own new-app placeholder", () => {
    expect(isPlaceholderApp(placeholder)).toBe(true);
    expect(isPlaceholderApp({ ...placeholder, slug: "new-app" })).toBe(true);
  });

  it("leaves a resumed app with a chosen name alone", () => {
    expect(isPlaceholderApp({ ...placeholder, slug: "storefront" })).toBe(false);
    expect(isPlaceholderApp({ ...placeholder, slug: "new-app-store" })).toBe(false);
  });
});

describe("findReusablePlaceholder", () => {
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

describe("appNameError", () => {
  const taken = new Set(["checkout-api"]);

  it.each([
    ["", "Enter a name"],
    ["  !! ", "Enter a name"],
    ["ab", "Use at least 3 characters"],
    ["a".repeat(256), "Use 255 characters or fewer"],
    ["Checkout API", "Another app in this project already has this name"],
  ])("rejects %j", (name, message) => {
    expect(appNameError(name, taken)).toBe(message);
  });

  it("accepts a free name of a valid length", () => {
    expect(appNameError("billing-worker", taken)).toBe(null);
    expect(appNameError("a".repeat(255), taken)).toBe(null);
  });
});
