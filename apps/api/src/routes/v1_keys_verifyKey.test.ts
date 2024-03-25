import { describe, expect, test } from "vitest";

import { ErrorResponse } from "@/pkg/errors";
import { eq, schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { RouteHarness } from "src/pkg/testutil/route-harness";
import { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "./v1_keys_verifyKey";

test("returns 200", async (t) => {
  const h = await RouteHarness.init(t);

  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await h.db.insert(schema.keys).values({
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: h.resources.userWorkspace.id,
    createdAt: new Date(),
  });

  const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: "/v1/keys.verifyKey",
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
  test("returns 400", async (t) => {
    const h = await RouteHarness.init(t);
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.insert(schema.keys).values({
      id: newId("key"),
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });

    const res = await h.post<any, ErrorResponse>({
      url: "/v1/keys.verifyKey",
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
    async (t) => {
      const h = await RouteHarness.init(t);
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.insert(schema.keys).values({
        id: newId("key"),
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        expires: new Date(Date.now() + 5000),
      });

      const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
        url: "/v1/keys.verifyKey",
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
      const secondResponse = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
        url: "/v1/keys.verifyKey",
        headers: {
          "Content-Type": "application/json",
        },
        body: {
          key,
          apiId: h.resources.userApi.id,
        },
      });
      expect(secondResponse.status).toEqual(200);
      expect(secondResponse.body.valid).toBe(false);
    },
    { timeout: 20000 },
  );
});

describe("with ip whitelist", () => {
  describe("with valid ip", () => {
    test("returns valid", async (t) => {
      const h = await RouteHarness.init(t);
      const keyAuthId = newId("keyAuth");
      await h.db.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
      });

      const apiId = newId("api");
      await h.db.insert(schema.apis).values({
        id: apiId,
        workspaceId: h.resources.userWorkspace.id,
        name: "test",
        authType: "key",
        keyAuthId: keyAuthId,
        ipWhitelist: JSON.stringify(["100.100.100.100"]),
        createdAt: new Date(),
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.insert(schema.keys).values({
        id: newId("key"),
        keyAuthId: keyAuthId,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
      });

      const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
        url: "/v1/keys.verifyKey",
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
    test(
      "returns invalid",
      async (t) => {
        const h = await RouteHarness.init(t);
        const keyAuthid = newId("keyAuth");
        await h.db.insert(schema.keyAuth).values({
          id: keyAuthid,
          workspaceId: h.resources.userWorkspace.id,
          createdAt: new Date(),
        });

        const apiId = newId("api");
        await h.db.insert(schema.apis).values({
          id: apiId,
          workspaceId: h.resources.userWorkspace.id,
          name: "test",
          authType: "key",
          keyAuthId: keyAuthid,
          ipWhitelist: JSON.stringify(["100.100.100.100"]),
          createdAt: new Date(),
        });

        const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
        await h.db.insert(schema.keys).values({
          id: newId("key"),
          keyAuthId: keyAuthid,
          hash: await sha256(key),
          start: key.slice(0, 8),
          workspaceId: h.resources.userWorkspace.id,
          createdAt: new Date(),
        });

        const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
          url: "/v1/keys.verifyKey",
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
      },
      { timeout: 20000 },
    );
  });
});

describe("with enabled key", () => {
  test("returns valid", async (t) => {
    const h = await RouteHarness.init(t);
    const keyAuthId = newId("keyAuth");
    await h.db.insert(schema.keyAuth).values({
      id: keyAuthId,
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });

    const apiId = newId("api");
    await h.db.insert(schema.apis).values({
      id: apiId,
      workspaceId: h.resources.userWorkspace.id,
      name: "test",
      authType: "key",
      keyAuthId: keyAuthId,
      createdAt: new Date(),
    });

    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.insert(schema.keys).values({
      id: newId("key"),
      keyAuthId: keyAuthId,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
      enabled: true,
    });

    const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
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

describe("with disabled key", () => {
  test("returns invalid", async (t) => {
    const h = await RouteHarness.init(t);
    const keyAuthid = newId("keyAuth");
    await h.db.insert(schema.keyAuth).values({
      id: keyAuthid,
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });

    const apiId = newId("api");
    await h.db.insert(schema.apis).values({
      id: apiId,
      workspaceId: h.resources.userWorkspace.id,
      name: "test",
      authType: "key",
      keyAuthId: keyAuthid,
      createdAt: new Date(),
    });

    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.insert(schema.keys).values({
      id: newId("key"),
      keyAuthId: keyAuthid,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
      enabled: false,
    });

    const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key,
        apiId: h.resources.userApi.id,
      },
    });
    expect(res.status).toEqual(200);
    expect(res.body.valid).toBe(false);
    expect(res.body.code).toEqual("DISABLED");
  });
});

test("returns the environment of a key", async (t) => {
  const h = await RouteHarness.init(t);

  const environment = "test";
  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await h.db.insert(schema.keys).values({
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: h.resources.userWorkspace.id,
    createdAt: new Date(),
    environment,
  });

  const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: "/v1/keys.verifyKey",
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
  expect(res.body.environment).toEqual(environment);
});

describe("disabled workspace", () => {
  test("should reject the request", async (t) => {
    const h = await RouteHarness.init(t);
    await h.db
      .update(schema.workspaces)
      .set({ enabled: false })
      .where(eq(schema.workspaces.id, h.resources.userWorkspace.id));

    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.insert(schema.keys).values({
      id: newId("key"),
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });
    const res = await h.post<V1KeysVerifyKeyRequest, ErrorResponse>({
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key,
        apiId: h.resources.userApi.id,
      },
    });
    expect(res.status).toEqual(403);
    expect(res.body).toMatchObject({
      error: {
        code: "FORBIDDEN",
        docs: "https://unkey.dev/docs/api-reference/errors/code/FORBIDDEN",
        message: "workspace is disabled",
      },
    });
  });
});
