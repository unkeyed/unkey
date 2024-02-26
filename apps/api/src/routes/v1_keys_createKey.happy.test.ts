import { afterEach, beforeEach, describe, expect, test } from "vitest";

import { sha256 } from "@unkey/hash";

import { RouteHarness } from "@/pkg/testutil/route-harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import {
  V1KeysCreateKeyRequest,
  V1KeysCreateKeyResponse,
  registerV1KeysCreateKey,
} from "./v1_keys_createKey";

let h: RouteHarness;
beforeEach(async () => {
  h = new RouteHarness();
  h.useRoutes(registerV1KeysCreateKey);
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
test("creates key", async () => {
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      byteLength: 16,
      apiId: h.resources.userApi.id,
      enabled: true,
    },
  });

  expect(res.status).toEqual(200);

  const found = await h.db.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.keyId),
  });
  expect(found).toBeDefined();
  expect(found!.hash).toEqual(await sha256(res.body.key));
});

describe("with enabled flag", () => {
  describe("not set", () => {
    test("should still create an enabled key", async () => {
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          byteLength: 16,
          apiId: h.resources.userApi.id,
        },
      });

      expect(res.status).toEqual(200);

      const found = await h.db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
      });
      expect(found).toBeDefined();
      expect(found!.hash).toEqual(await sha256(res.body.key));
      expect(found!.enabled).toBe(true);
    });
  });
  describe("enabled: false", () => {
    test("should create a disabled key", async () => {
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          byteLength: 16,
          apiId: h.resources.userApi.id,
          enabled: false,
        },
      });

      expect(res.status).toEqual(200);

      const found = await h.db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
      });
      expect(found).toBeDefined();
      expect(found!.hash).toEqual(await sha256(res.body.key));
      expect(found!.enabled).toBe(false);
    });
  });
  describe("enabled: true", () => {
    test("should create an enabled key", async () => {
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          byteLength: 16,
          apiId: h.resources.userApi.id,
          enabled: true,
        },
      });

      expect(res.status).toEqual(200);

      const found = await h.db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
      });
      expect(found).toBeDefined();
      expect(found!.hash).toEqual(await sha256(res.body.key));
      expect(found!.enabled).toBe(true);
    });
  });
});

describe("with prefix", () => {
  test("start includes prefix", async () => {
    const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

    const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        byteLength: 16,
        apiId: h.resources.userApi.id,
        prefix: "prefix",
        enabled: true,
      },
    });

    expect(res.status).toEqual(200);

    const key = await h.db.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
    });
    expect(key).toBeDefined();
    expect(key!.start.startsWith("prefix_")).toBe(true);
  });
});

describe("roles", () => {
  test("connects the specified roles", async () => {
    const roles = ["r1", "r2"];
    await h.db.insert(schema.roles).values(
      roles.map((name) => ({
        id: newId("role"),
        name,
        workspaceId: h.resources.userWorkspace.id,
      })),
    );

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

    const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        apiId: h.resources.userApi.id,
        roles,
      },
    });

    expect(res.status).toEqual(200);

    const key = await h.db.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
      with: {
        roles: {
          with: {
            role: true,
          },
        },
      },
    });
    expect(key).toBeDefined();
    expect(key!.roles.length).toBe(2);
    for (const r of key!.roles!) {
      expect(roles).include(r.role.name);
    }
  });
});

test("creates a key with environment", async () => {
  const environment = "test";

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      environment,
    },
  });

  expect(res.status).toEqual(200);

  const key = await h.db.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.keyId),
  });
  expect(key).toBeDefined();
  expect(key!.environment).toBe(environment);
});
