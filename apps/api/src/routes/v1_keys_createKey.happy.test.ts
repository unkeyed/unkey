import { describe, expect, test } from "vitest";

import { sha256 } from "@unkey/hash";

import { RouteHarness } from "@/pkg/testutil/route-harness";
import {
  V1KeysCreateKeyRequest,
  V1KeysCreateKeyResponse,
  registerV1KeysCreateKey,
} from "./v1_keys_createKey";

test("creates key", async () => {
  using h = new RouteHarness();
  await h.seed();
  h.useRoutes(registerV1KeysCreateKey);

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
      using h = new RouteHarness();
      await h.seed();
      h.useRoutes(registerV1KeysCreateKey);

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
      using h = new RouteHarness();
      await h.seed();
      h.useRoutes(registerV1KeysCreateKey);
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
      using h = new RouteHarness();
      await h.seed();
      h.useRoutes(registerV1KeysCreateKey);
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
    using h = new RouteHarness();
    await h.seed();
    h.useRoutes(registerV1KeysCreateKey);
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
