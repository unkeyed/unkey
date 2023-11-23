import { describe, expect, test } from "bun:test";

import { ErrorResponse } from "@/pkg/errors";
import { init } from "@/pkg/global";
import { newApp } from "@/pkg/hono/app";
import { testEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { seed } from "@/pkg/testutil/seed";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import {
  LegacyKeysVerifyKeyRequest,
  LegacyKeysVerifyKeyResponse,
  registerLegacyKeysVerifyKey,
} from "./legacy_keys_verifyKey";

test("returns 200", async () => {
  const env = testEnv();
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerLegacyKeysVerifyKey(app);

  const r = await seed(env);

  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await r.database.insert(schema.keys).values({
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: r.userWorkspace.id,
    createdAt: new Date(),
  });

  const res = await fetchRoute<LegacyKeysVerifyKeyRequest, LegacyKeysVerifyKeyResponse>(app, {
    method: "POST",
    url: "/v1/keys/verify",
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      key,
      apiId: r.userApi.id,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body.valid).toBeTrue();
});

describe("bad request", () => {
  test("returns 400", async () => {
    const env = testEnv();
    // @ts-ignore
    init({ env });
    const app = newApp();
    registerLegacyKeysVerifyKey(app);

    const r = await seed(env);

    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await r.database.insert(schema.keys).values({
      id: newId("key"),
      keyAuthId: r.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: r.userWorkspace.id,
      createdAt: new Date(),
    });

    const res = await fetchRoute<any, ErrorResponse>(app, {
      method: "POST",
      url: "/v1/keys/verify",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        something: "else",
      },
    });

    expect(res.status).toEqual(400);
  });
});

describe("with temporary key", () => {
  test("returns valid", async () => {
    const env = testEnv();
    // @ts-ignore
    init({ env });
    const app = newApp();
    registerLegacyKeysVerifyKey(app);

    const r = await seed(env);

    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await r.database.insert(schema.keys).values({
      id: newId("key"),
      keyAuthId: r.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: r.userWorkspace.id,
      createdAt: new Date(),
      expires: new Date(Date.now() + 5000),
    });

    const res = await fetchRoute<LegacyKeysVerifyKeyRequest, LegacyKeysVerifyKeyResponse>(app, {
      method: "POST",
      url: "/v1/keys/verify",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key,
        apiId: r.userApi.id,
      },
    });
    expect(res.status).toEqual(200);
    expect(res.body.valid).toBeTrue();

    await new Promise((resolve) => setTimeout(resolve, 6000));
    const secondResponse = await fetchRoute<
      LegacyKeysVerifyKeyRequest,
      LegacyKeysVerifyKeyResponse
    >(app, {
      method: "POST",
      url: "/v1/keys/verify",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key,
        apiId: r.userApi.id,
      },
    });
    expect(secondResponse.status).toEqual(200);
    expect(secondResponse.body.valid).toBeFalse();
  });
});

describe("with ip whitelist", () => {
  describe("with valid ip", () => {
    test("returns valid", async () => {
      const env = testEnv();
      // @ts-ignore
      init({ env });
      const app = newApp();
      registerLegacyKeysVerifyKey(app);

      const r = await seed(env);

      const keyAuthId = newId("keyAuth");
      await r.database.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: r.userWorkspace.id,
      });

      const apiId = newId("api");
      await r.database.insert(schema.apis).values({
        id: apiId,
        workspaceId: r.userWorkspace.id,
        name: "test",
        authType: "key",
        keyAuthId: keyAuthId,
        ipWhitelist: JSON.stringify(["100.100.100.100"]),
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await r.database.insert(schema.keys).values({
        id: newId("key"),
        keyAuthId: keyAuthId,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: r.userWorkspace.id,
        createdAt: new Date(),
      });

      const res = await fetchRoute<LegacyKeysVerifyKeyRequest, LegacyKeysVerifyKeyResponse>(app, {
        method: "POST",
        url: "/v1/keys/verify",
        headers: {
          "Content-Type": "application/json",
          "True-Client-IP": "100.100.100.100",
        },
        body: {
          key,
          apiId,
        },
      });
      expect(res.status).toEqual(200);
      expect(res.body.valid).toBeTrue();
    });
  });
  describe("with invalid ip", () => {
    test("returns invalid", async () => {
      const env = testEnv();
      // @ts-ignore
      init({ env });
      const app = newApp();
      registerLegacyKeysVerifyKey(app);

      const r = await seed(env);

      const keyAuthid = newId("keyAuth");
      await r.database.insert(schema.keyAuth).values({
        id: keyAuthid,
        workspaceId: r.userWorkspace.id,
      });

      const apiId = newId("api");
      await r.database.insert(schema.apis).values({
        id: apiId,
        workspaceId: r.userWorkspace.id,
        name: "test",
        authType: "key",
        keyAuthId: keyAuthid,
        ipWhitelist: JSON.stringify(["100.100.100.100"]),
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await r.database.insert(schema.keys).values({
        id: newId("key"),
        keyAuthId: keyAuthid,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: r.userWorkspace.id,
        createdAt: new Date(),
      });

      const res = await fetchRoute<LegacyKeysVerifyKeyRequest, LegacyKeysVerifyKeyResponse>(app, {
        method: "POST",
        url: "/v1/keys/verify",
        headers: {
          "Content-Type": "application/json",
          "True-Client-IP": "200.200.200.200",
        },
        body: {
          key,
          apiId: r.userApi.id,
        },
      });
      expect(res.status).toEqual(200);
      expect(res.body.valid).toBeFalse();
      expect(res.body.code).toEqual("FORBIDDEN");
    });
  });
});
