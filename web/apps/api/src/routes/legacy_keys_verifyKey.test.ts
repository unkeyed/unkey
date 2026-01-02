import { describe, expect, test } from "vitest";

import type { ErrorResponse } from "@/pkg/errors";

import { eq, schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type {
  LegacyKeysVerifyKeyRequest,
  LegacyKeysVerifyKeyResponse,
} from "./legacy_keys_verifyKey";

test("returns 200", async (t) => {
  const h = await IntegrationHarness.init(t);
  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await h.db.primary.insert(schema.keys).values({
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: h.resources.userWorkspace.id,
    createdAtM: Date.now(),
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.valid).toBe(true);
});

describe("bad request", () => {
  test("returns 400", async (t) => {
    const h = await IntegrationHarness.init(t);
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: newId("test"),
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAtM: Date.now(),
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

describe("disabled workspace", () => {
  test("returns 403", async (t) => {
    const h = await IntegrationHarness.init(t);
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: newId("test"),
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAtM: Date.now(),
    });
    await h.db.primary
      .update(schema.workspaces)
      .set({
        enabled: false,
      })
      .where(eq(schema.workspaces.id, h.resources.userWorkspace.id));

    const res = await h.post<any, ErrorResponse>({
      url: "/v1/keys/verify",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key: key,
      },
    });

    expect(res.status).toEqual(403);
  });
});

describe("with temporary key", () => {
  test("returns valid", async (t) => {
    const h = await IntegrationHarness.init(t);
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: newId("test"),
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAtM: Date.now(),
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
    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
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
  });
});

describe("with ip whitelist", () => {
  describe("with valid ip", () => {
    test("returns valid", async (t) => {
      const h = await IntegrationHarness.init(t);
      const keyAuthId = newId("test");
      await h.db.primary.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: h.resources.userWorkspace.id,
        createdAtM: Date.now(),
        deletedAtM: null,
      });

      const apiId = newId("api");
      await h.db.primary.insert(schema.apis).values({
        id: apiId,
        workspaceId: h.resources.userWorkspace.id,
        name: "test",
        authType: "key",
        keyAuthId: keyAuthId,
        ipWhitelist: ["100.100.100.100"].join(","),
        createdAtM: Date.now(),
        deletedAtM: null,
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: newId("test"),
        keyAuthId: keyAuthId,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAtM: Date.now(),
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
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
      expect(res.body.valid).toBe(true);
    });
  });
  describe("with invalid ip", () => {
    test("returns invalid", async (t) => {
      const h = await IntegrationHarness.init(t);
      const keyAuthid = newId("test");
      await h.db.primary.insert(schema.keyAuth).values({
        id: keyAuthid,
        workspaceId: h.resources.userWorkspace.id,
        createdAtM: Date.now(),
        deletedAtM: null,
      });

      const apiId = newId("test");
      await h.db.primary.insert(schema.apis).values({
        id: apiId,
        workspaceId: h.resources.userWorkspace.id,
        name: "test",
        authType: "key",
        keyAuthId: keyAuthid,
        ipWhitelist: ["100.100.100.100"].join(","),
        createdAtM: Date.now(),
        deletedAtM: null,
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: newId("test"),
        keyAuthId: keyAuthid,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAtM: Date.now(),
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
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
      expect(res.body.valid).toBe(false);
      expect(res.body.code).toEqual("FORBIDDEN");
    });
  });
});
