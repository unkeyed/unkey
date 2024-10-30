import { describe, expect, test } from "vitest";

import type { ErrorResponse } from "@/pkg/errors";
import { eq, schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "./v1_keys_verifyKey";

test("returns 200", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await h.db.primary.insert(schema.keys).values({
    id: newId("test"),
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
      const h = await IntegrationHarness.init(t);
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: newId("test"),
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        expires: new Date(Date.now() + 2000),
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
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
      expect(res.body.valid).toBe(true);

      await new Promise((resolve) => setTimeout(resolve, 2500));
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

  test(
    "returns all data",
    async (t) => {
      const h = await IntegrationHarness.init(t);

      const now = new Date();

      const secret = new KeyV1({ prefix: "test", byteLength: 16 }).toString();

      const key = {
        id: newId("test"),
        name: "hello world",
        ownerId: newId("test"),
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(secret),
        start: secret.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: now,
        expires: now,
        environment: "prod",
        meta: JSON.stringify({ hello: "world" }),
      };

      await h.db.primary.insert(schema.keys).values(key);

      const permission = {
        id: newId("test"),
        workspaceId: h.resources.userWorkspace.id,
        name: "permission",
      };
      await h.db.primary.insert(schema.permissions).values(permission);
      await h.db.primary.insert(schema.keysPermissions).values({
        keyId: key.id,
        permissionId: permission.id,
        workspaceId: h.resources.userWorkspace.id,
      });

      await new Promise((resolve) => setTimeout(resolve, 2500));
      const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
        url: "/v1/keys.verifyKey",
        headers: {
          "Content-Type": "application/json",
        },
        body: {
          key: secret,
          apiId: h.resources.userApi.id,
        },
      });
      expect(res.status).toEqual(200);
      expect(res.body.valid).toBe(false);
      expect(res.body.code).toBe("EXPIRED");
      expect(res.body.meta).toMatchObject({ hello: "world" });
      expect(res.body.expires).toBe(key.expires.getTime());
      expect(res.body.environment).toBe(key.environment);
      expect(res.body.name).toBe(key.name);
      expect(res.body.ownerId).toBe(key.ownerId);
      expect(res.body.permissions).toMatchObject([permission.name]);
    },
    { timeout: 20000 },
  );
});

describe("with metadata", () => {
  test(
    "returns meta when key is disabled",
    async (t) => {
      const h = await IntegrationHarness.init(t);
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: newId("test"),
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        meta: JSON.stringify({
          disabledReason: "cause I can",
        }),
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
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
      expect(res.body.valid).toBe(false);
      expect(res.body.meta).toMatchObject({ disabledReason: "cause I can" });
    },
    { timeout: 20000 },
  );
});

describe("with identity", () => {
  test(
    "returns the identity",
    async (t) => {
      const h = await IntegrationHarness.init(t);

      const identity = {
        id: newId("test"),
        externalId: randomUUID(),
        workspaceId: h.resources.userWorkspace.id,
      };
      await h.db.primary.insert(schema.identities).values(identity);

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: newId("test"),
        keyAuthId: h.resources.userKeyAuth.id,
        identityId: identity.id,
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
          apiId: h.resources.userApi.id,
        },
      });
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
      expect(res.body.valid).toBe(true);
      expect(res.body.identity).toMatchObject({
        id: identity.id,
        externalId: identity.externalId,
      });
    },
    { timeout: 20000 },
  );
  test(
    "returns the meta",
    async (t) => {
      const h = await IntegrationHarness.init(t);

      const identity = {
        id: newId("test"),
        externalId: randomUUID(),
        workspaceId: h.resources.userWorkspace.id,
        meta: {
          hello: "world",
        },
      };
      await h.db.primary.insert(schema.identities).values(identity);

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: newId("test"),
        keyAuthId: h.resources.userKeyAuth.id,
        identityId: identity.id,
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
          apiId: h.resources.userApi.id,
        },
      });
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
      expect(res.body.valid).toBe(true);
      expect(res.body.identity).toMatchObject({
        id: identity.id,
        externalId: identity.externalId,
        meta: identity.meta,
      });
    },
    { timeout: 20000 },
  );
});

describe("when ratelimited", () => {
  test("returns ownerId and identity ", async (t) => {
    const h = await IntegrationHarness.init(t);
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    const externalId = newId("test");

    const identityId = newId("test");
    await h.db.primary.insert(schema.identities).values({
      id: identityId,
      externalId,
      workspaceId: h.resources.userWorkspace.id,
    });

    await h.db.primary.insert(schema.keys).values({
      id: newId("test"),
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      identityId,
      ownerId: externalId,
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
      ratelimitAsync: false,
      ratelimitLimit: 0,
      ratelimitDuration: 60_000,
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
    expect(res.body.code).toEqual("RATE_LIMITED");
    expect(res.body.ownerId).toEqual(externalId);
    expect(res.body.identity).toBeDefined();
    expect(res.body.identity!.externalId).toEqual(externalId);
  });
});

describe("with ratelimit override", () => {
  test("deducts the correct number of tokens", { timeout: 20000 }, async (t) => {
    const h = await IntegrationHarness.init(t);
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: newId("test"),
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
      ratelimitLimit: 10,
      ratelimitDuration: 60_000,
      ratelimitAsync: false,
    });

    const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key,
        apiId: h.resources.userApi.id,
        ratelimit: {
          cost: 4,
        },
      },
    });
    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    expect(res.body.valid).toBe(true);
    // expect(res.body.ratelimit?.remaining).toEqual(6);
  });
});

describe("with default ratelimit", () => {
  test("uses the on-key defined settings", { timeout: 20000 }, async (t) => {
    const h = await IntegrationHarness.init(t);
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: newId("test"),
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
      ratelimitLimit: 10,
      ratelimitDuration: 60_000,
      ratelimitAsync: false,
    });

    const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key,
        apiId: h.resources.userApi.id,
        ratelimits: [
          {
            name: "default",
          },
        ],
      },
    });
    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    expect(res.body.valid).toBe(true);
    expect(res.body.ratelimit).toBeDefined();
    expect(res.body.ratelimit!.limit).toEqual(10);
  });
});

describe("with ratelimit", () => {
  describe("with valid key", () => {
    test.skip(
      "returns the limit ",
      async (t) => {
        const h = await IntegrationHarness.init(t);
        const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
        await h.db.primary.insert(schema.keys).values({
          id: newId("test"),
          keyAuthId: h.resources.userKeyAuth.id,
          hash: await sha256(key),
          start: key.slice(0, 8),
          workspaceId: h.resources.userWorkspace.id,
          createdAt: new Date(),
          ratelimitLimit: 10,
          ratelimitDuration: 60_000,
          ratelimitAsync: false,
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
        expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
        expect(res.body.valid).toBe(true);
        expect(res.body.ratelimit).toBeDefined();
        expect(res.body.ratelimit!.limit).toEqual(10);
        expect(res.body.ratelimit!.remaining).toEqual(9);
        expect(res.body.ratelimit!.reset).toBeGreaterThan(Date.now() - 60_000);
      },
      { timeout: 20000 },
    );
  });
  describe("with used up key", () => {
    test(
      "returns valid=false and correct code ",
      async (t) => {
        const h = await IntegrationHarness.init(t);
        const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
        await h.db.primary.insert(schema.keys).values({
          id: newId("test"),
          keyAuthId: h.resources.userKeyAuth.id,
          hash: await sha256(key),
          start: key.slice(0, 8),
          workspaceId: h.resources.userWorkspace.id,
          createdAt: new Date(),
          remaining: 0,
          ratelimitLimit: 10,
          ratelimitDuration: 60_000,
          ratelimitAsync: false,
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
        expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
        expect(res.body.valid).toBe(false);
        expect(res.body.code).toBe("USAGE_EXCEEDED");
      },
      { timeout: 20000 },
    );
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
        createdAt: new Date(),
      });

      const apiId = newId("api");
      await h.db.primary.insert(schema.apis).values({
        id: apiId,
        workspaceId: h.resources.userWorkspace.id,
        name: "test",
        authType: "key",
        keyAuthId: keyAuthId,
        ipWhitelist: ["100.100.100.100"].join(","),
        createdAt: new Date(),
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: newId("test"),
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
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
      expect(res.body.valid).toBe(true);
    });
  });
  describe("with invalid ip", () => {
    test(
      "returns invalid",
      async (t) => {
        const h = await IntegrationHarness.init(t);
        const keyAuthid = newId("test");
        await h.db.primary.insert(schema.keyAuth).values({
          id: keyAuthid,
          workspaceId: h.resources.userWorkspace.id,
          createdAt: new Date(),
        });

        const apiId = newId("api");
        await h.db.primary.insert(schema.apis).values({
          id: apiId,
          workspaceId: h.resources.userWorkspace.id,
          name: "test",
          authType: "key",
          keyAuthId: keyAuthid,
          ipWhitelist: ["100.100.100.100"].join(","),
          createdAt: new Date(),
        });

        const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
        await h.db.primary.insert(schema.keys).values({
          id: newId("test"),
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
        expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
        expect(res.body.valid).toBe(false);
        expect(res.body.code).toEqual("FORBIDDEN");
      },
      { timeout: 20000 },
    );
  });
});

describe("with enabled key", () => {
  test("returns valid", async (t) => {
    const h = await IntegrationHarness.init(t);
    const keyAuthId = newId("test");
    await h.db.primary.insert(schema.keyAuth).values({
      id: keyAuthId,
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });

    const apiId = newId("api");
    await h.db.primary.insert(schema.apis).values({
      id: apiId,
      workspaceId: h.resources.userWorkspace.id,
      name: "test",
      authType: "key",
      keyAuthId: keyAuthId,
      createdAt: new Date(),
    });

    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: newId("test"),
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
    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    expect(res.body.valid).toBe(true);
  });
});

describe("with disabled key", () => {
  test("returns invalid", async (t) => {
    const h = await IntegrationHarness.init(t);
    const keyAuthid = newId("test");
    await h.db.primary.insert(schema.keyAuth).values({
      id: keyAuthid,
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });

    const apiId = newId("api");
    await h.db.primary.insert(schema.apis).values({
      id: apiId,
      workspaceId: h.resources.userWorkspace.id,
      name: "test",
      authType: "key",
      keyAuthId: keyAuthid,
      createdAt: new Date(),
    });

    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: newId("test"),
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
    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    expect(res.body.valid).toBe(false);
    expect(res.body.code).toEqual("DISABLED");
  });
});

test("returns the environment of a key", async (t) => {
  const h = await IntegrationHarness.init(t);

  const environment = "test";
  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await h.db.primary.insert(schema.keys).values({
    id: newId("test"),
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
  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.valid).toBe(true);
  expect(res.body.environment).toEqual(environment);
});

describe("disabled workspace", () => {
  test("should reject the request", async (t) => {
    const h = await IntegrationHarness.init(t);
    await h.db.primary
      .update(schema.workspaces)
      .set({ enabled: false })
      .where(eq(schema.workspaces.id, h.resources.userWorkspace.id));

    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: newId("test"),
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

describe("key is soft deleted", () => {
  test("returns NOT_FOUND", async (t) => {
    const h = await IntegrationHarness.init(t);
    const key = await h.createKey();
    await h.db.primary
      .update(schema.keys)
      .set({ deletedAt: new Date() })
      .where(eq(schema.keys.id, key.keyId));

    const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key: key.key,
        apiId: h.resources.userApi.id,
      },
    });
    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    expect(res.body.valid).toBe(false);
    expect(res.body.code).toBe("NOT_FOUND");
  });
});

describe("key exists but keyspace is soft deleted", () => {
  test("returns NOT_FOUND", async (t) => {
    const h = await IntegrationHarness.init(t);
    const key = await h.createKey();

    await h.db.primary
      .update(schema.keyAuth)
      .set({ deletedAt: new Date() })
      .where(eq(schema.keyAuth.id, h.resources.userKeyAuth.id));

    const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key: key.key,
        apiId: h.resources.userApi.id,
      },
    });
    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    expect(res.body.valid).toBe(false);
    expect(res.body.code).toBe("NOT_FOUND");
  });
});

describe("key exists but api is soft deleted", () => {
  test("returns NOT_FOUND", async (t) => {
    const h = await IntegrationHarness.init(t);
    const key = await h.createKey();

    await h.db.primary
      .update(schema.apis)
      .set({ deletedAt: new Date() })
      .where(eq(schema.apis.id, h.resources.userApi.id));

    const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key: key.key,
        apiId: h.resources.userApi.id,
      },
    });
    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    expect(res.body.valid).toBe(false);
    expect(res.body.code).toBe("NOT_FOUND");
  });
});

describe("key exists but api is hard deleted", () => {
  test("returns NOT_FOUND", async (t) => {
    const h = await IntegrationHarness.init(t);
    const key = await h.createKey();

    await h.db.primary.delete(schema.apis).where(eq(schema.apis.id, h.resources.userApi.id));

    const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key: key.key,
        apiId: h.resources.userApi.id,
      },
    });
    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    expect(res.body.valid).toBe(false);
    expect(res.body.code).toBe("NOT_FOUND");
  });
});
