import { describe } from "node:test";
import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { type Flatten, Unkey, and, or } from "@unkey/api/src/index"; // use unbundled raw esm typescript
import { afterEach, beforeEach, expect, test } from "vitest";

let h: IntegrationHarness;

beforeEach(async () => {
  h = new IntegrationHarness();
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
test("with raw query", async () => {
  await h.seed();

  const { key } = await h.createKey();

  const sdk = new Unkey({
    baseUrl: h.baseUrl,
    rootKey: "not-needed-for-this-test",
  });

  const { result, error } = await sdk.keys.verify({
    apiId: h.resources.userApi.id,
    key,
    authorization: {
      permissions: {
        and: ["p1", "p2"],
      },
    },
  });
  expect(error).toBeUndefined();
  expect(result).toBeDefined();

  expect(result!.valid).toBe(false);
  expect(result!.code).toBe("INSUFFICIENT_PERMISSIONS");
});

describe("with typesafe generated permissions", () => {
  test("returns valid", async () => {
    await h.seed();

    const { key } = await h.createKey({
      roles: [
        {
          name: "domain.manager",
          permissions: ["domain.create", "dns.record.create", "domain.delete"],
        },
      ],
    });

    const sdk = new Unkey({
      baseUrl: h.baseUrl,
      rootKey: "not-needed-for-this-test",
    });

    type Resources = {
      domain: "create" | "delete" | "read";
      dns: {
        record: "create" | "read" | "delete";
      };
    };
    type Permissions = Flatten<Resources, ".">;

    const { result, error } = await sdk.keys.verify<Permissions>({
      apiId: h.resources.userApi.id,
      key,
      authorization: {
        permissions: or("domain.create", "dns.record.create"),
      },
    });

    expect(error).toBeUndefined();
    expect(result).toBeDefined();

    expect(result!.valid).toBe(true);
  });

  test("with helper functions", async () => {
    await h.seed();

    const { key } = await h.createKey();

    const sdk = new Unkey({
      baseUrl: h.baseUrl,
      rootKey: "not-needed-for-this-test",
    });

    type Permission = "p1" | "p2";

    const { result, error } = await sdk.keys.verify<Permission>({
      apiId: h.resources.userApi.id,
      key,
      authorization: {
        permissions: and("p1", "p2"),
      },
    });
    expect(error).toBeUndefined();
    expect(result).toBeDefined();

    expect(result!.valid).toBe(false);
    expect(result!.code).toBe("INSUFFICIENT_PERMISSIONS");
  });
});
