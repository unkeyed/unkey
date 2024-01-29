import { describe, expect, test } from "vitest";

import { ErrorResponse } from "@/pkg/errors";

import { Harness } from "@/pkg/testutil/harness";
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
  const h = await Harness.init();
  h.useRoutes(registerLegacyKeysVerifyKey);

  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await h.resources.database.insert(schema.keys).values({
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: h.resources.userWorkspace.id,
    createdAt: new Date(),
  });

  const res = await h.post<LegacyKeysVerifyKeyRequest, LegacyKeysVerifyKeyResponse>({
    url: "/v1/keys/verify",
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      key,
      apiId: h.resources.userApi.id,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body.valid).toBe(true);
});

describe("bad request", () => {
  test("returns 400", async () => {
    const h = await Harness.init();
    h.useRoutes(registerLegacyKeysVerifyKey);

    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.resources.database.insert(schema.keys).values({
      id: newId("key"),
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });

    const res = await h.post<any, ErrorResponse>({
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
  test(
    "returns valid",
    async () => {
      const h = await Harness.init();
      h.useRoutes(registerLegacyKeysVerifyKey);

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.resources.database.insert(schema.keys).values({
        id: newId("key"),
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        expires: new Date(Date.now() + 5000),
      });

      const res = await h.post<LegacyKeysVerifyKeyRequest, LegacyKeysVerifyKeyResponse>({
        url: "/v1/keys/verify",
        headers: {
          "Content-Type": "application/json",
        },
        body: {
          key,
          apiId: h.resources.userApi.id,
        },
      });
      expect(res.status).toEqual(200);
      expect(res.body.valid).toBe(true);

      await new Promise((resolve) => setTimeout(resolve, 6000));
      const secondResponse = await h.post<LegacyKeysVerifyKeyRequest, LegacyKeysVerifyKeyResponse>({
        url: "/v1/keys/verify",
        headers: {
          "Content-Type": "application/json",
        },
        body: {
          key,
          apiId: h.resources.userApi.id,
        },
      });
      expect(secondResponse.status).toEqual(404);
      expect(secondResponse.body.valid).toBe(false);
    },
    { timeout: 10_000 },
  );
});

describe("with ip whitelist", () => {
  describe("with valid ip", () => {
    test("returns valid", async () => {
      const h = await Harness.init();
      h.useRoutes(registerLegacyKeysVerifyKey);

      const keyAuthId = newId("keyAuth");
      await h.resources.database.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        deletedAt: null,
      });

      const apiId = newId("api");
      await h.resources.database.insert(schema.apis).values({
        id: apiId,
        workspaceId: h.resources.userWorkspace.id,
        name: "test",
        authType: "key",
        keyAuthId: keyAuthId,
        ipWhitelist: JSON.stringify(["100.100.100.100"]),
        createdAt: new Date(),
        deletedAt: null,
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.resources.database.insert(schema.keys).values({
        id: newId("key"),
        keyAuthId: keyAuthId,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
      });

      const res = await h.post<LegacyKeysVerifyKeyRequest, LegacyKeysVerifyKeyResponse>({
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
      expect(res.body.valid).toBe(true);
    });
  });
  describe("with invalid ip", () => {
    test("returns invalid", async () => {
      const h = await Harness.init();
      h.useRoutes(registerLegacyKeysVerifyKey);

      const keyAuthid = newId("keyAuth");
      await h.resources.database.insert(schema.keyAuth).values({
        id: keyAuthid,
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        deletedAt: null,
      });

      const apiId = newId("api");
      await h.resources.database.insert(schema.apis).values({
        id: apiId,
        workspaceId: h.resources.userWorkspace.id,
        name: "test",
        authType: "key",
        keyAuthId: keyAuthid,
        ipWhitelist: JSON.stringify(["100.100.100.100"]),
        createdAt: new Date(),
        deletedAt: null,
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.resources.database.insert(schema.keys).values({
        id: newId("key"),
        keyAuthId: keyAuthid,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
      });

      const res = await h.post<LegacyKeysVerifyKeyRequest, LegacyKeysVerifyKeyResponse>({
        url: "/v1/keys/verify",
        headers: {
          "Content-Type": "application/json",
          "True-Client-IP": "200.200.200.200",
        },
        body: {
          key,
          apiId: h.resources.userApi.id,
        },
      });
      expect(res.status).toEqual(200);
      expect(res.body.valid).toBe(false);
      expect(res.body.code).toEqual("FORBIDDEN");
    });
  });
});
