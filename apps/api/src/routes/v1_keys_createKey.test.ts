import { describe, expect, test } from "vitest";

import { sha256 } from "@unkey/hash";

import { ErrorResponse } from "@/pkg/errors";
import { Harness } from "@/pkg/testutil/harness";
import {
  V1KeysCreateKeyRequest,
  V1KeysCreateKeyResponse,
  registerV1KeysCreateKey,
} from "./v1_keys_createKey";

describe("simple", () => {
  test("creates key", async () => {
    const h = await Harness.init();
    h.useRoutes(registerV1KeysCreateKey);

    const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${h.resources.rootKey}`,
      },
      body: {
        byteLength: 16,
        apiId: h.resources.userApi.id,
        enabled: true,
      },
    });

    expect(res.status).toEqual(200);

    const found = await h.resources.database.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
    });
    expect(found).toBeDefined();
    expect(found!.hash).toEqual(await sha256(res.body.key));
  });
});

describe("enabled", () => {
  describe("not set", () => {
    test("should still create an enabled key", async () => {
      const h = await Harness.init();
      h.useRoutes(registerV1KeysCreateKey);

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${h.resources.rootKey}`,
        },
        body: {
          byteLength: 16,
          apiId: h.resources.userApi.id,
        },
      });

      expect(res.status).toEqual(200);

      const found = await h.resources.database.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
      });
      expect(found).toBeDefined();
      expect(found!.hash).toEqual(await sha256(res.body.key));
      expect(found!.enabled).toBe(true);
    });
  });
  describe("false", () => {
    test("should create a disabled key", async () => {
      const h = await Harness.init();
      h.useRoutes(registerV1KeysCreateKey);

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${h.resources.rootKey}`,
        },
        body: {
          byteLength: 16,
          apiId: h.resources.userApi.id,
          enabled: false,
        },
      });

      expect(res.status).toEqual(200);

      const found = await h.resources.database.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
      });
      expect(found).toBeDefined();
      expect(found!.hash).toEqual(await sha256(res.body.key));
      expect(found!.enabled).toBe(false);
    });
  });
  describe("true", () => {
    test("should create an enabled key", async () => {
      const h = await Harness.init();
      h.useRoutes(registerV1KeysCreateKey);

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${h.resources.rootKey}`,
        },
        body: {
          byteLength: 16,
          apiId: h.resources.userApi.id,
          enabled: true,
        },
      });

      expect(res.status).toEqual(200);

      const found = await h.resources.database.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
      });
      expect(found).toBeDefined();
      expect(found!.hash).toEqual(await sha256(res.body.key));
      expect(found!.enabled).toBe(true);
    });
  });
});
describe("wrong ratelimit type", () => {
  test("reject the request", async () => {
    const h = await Harness.init();
    h.useRoutes(registerV1KeysCreateKey);

    const res = await h.post<V1KeysCreateKeyRequest, ErrorResponse>({
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${h.resources.rootKey}`,
      },
      body: {
        byteLength: 16,
        apiId: h.resources.userApi.id,
        ratelimit: {
          // @ts-expect-error
          type: "x",
        },
      },
    });

    expect(res.status).toEqual(400);
    expect(res.body.error.code).toEqual("BAD_REQUEST");
  });
});

describe("with prefix", () => {
  test("start includes prefix", async () => {
    const h = await Harness.init();
    h.useRoutes(registerV1KeysCreateKey);

    const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${h.resources.rootKey}`,
      },
      body: {
        byteLength: 16,
        apiId: h.resources.userApi.id,
        prefix: "prefix",
        enabled: true,
      },
    });

    expect(res.status).toEqual(200);

    const key = await h.resources.database.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
    });
    expect(key).toBeDefined();
    expect(key!.start.startsWith("prefix_")).toBe(true);
  });
});
