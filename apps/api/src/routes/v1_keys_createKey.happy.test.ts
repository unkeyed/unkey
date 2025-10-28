import { describe, expect, test } from "vitest";

import { sha256 } from "@unkey/hash";

import { eq, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { KeyV1 } from "@unkey/keys";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "./v1_keys_createKey";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "./v1_keys_verifyKey";

test("creates key", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      byteLength: 18,
      apiId: h.resources.userApi.id,
      enabled: true,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.keyId),
  });
  expect(found).toBeDefined();
  expect(found!.hash).toEqual(await sha256(res.body.key));
});

test("creates key with default prefix and bytes from keyAuth", async (t) => {
  const expectedPrefix = "_prefix";
  const expectedBytes = 66;

  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  await h.db.primary
    .update(schema.keyAuth)
    .set({
      defaultPrefix: expectedPrefix,
      defaultBytes: expectedBytes,
    })
    .where(eq(schema.keyAuth.id, h.resources.userKeyAuth.id));

  // Make the request without specifying prefix or byteLength
  const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      enabled: true,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.keyId),
  });

  const referenceKey = new KeyV1({
    byteLength: expectedBytes,
    prefix: expectedPrefix,
  }).toString();

  expect(found).toBeDefined();
  expect(found!.start).toEqual(referenceKey.slice(0, 5));
  expect(found!.hash).toEqual(await sha256(res.body.key));
});

describe("with enabled flag", () => {
  describe("not set", () => {
    test("should still create an enabled key", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          byteLength: 32,
          apiId: h.resources.userApi.id,
        },
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const found = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
      });
      expect(found).toBeDefined();
      expect(found!.hash).toEqual(await sha256(res.body.key));
      expect(found!.enabled).toBe(true);
    });
  });
  describe("enabled: false", () => {
    test("should create a disabled key", async (t) => {
      const h = await IntegrationHarness.init(t);
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

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const found = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
      });
      expect(found).toBeDefined();
      expect(found!.hash).toEqual(await sha256(res.body.key));
      expect(found!.enabled).toBe(false);
    });
  });
  describe("enabled: true", () => {
    test("should create an enabled key", async (t) => {
      const h = await IntegrationHarness.init(t);
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

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const found = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
      });
      expect(found).toBeDefined();
      expect(found!.hash).toEqual(await sha256(res.body.key));
      expect(found!.enabled).toBe(true);
    });
  });
});

describe("with prefix", () => {
  test("start includes prefix", async (t) => {
    const h = await IntegrationHarness.init(t);
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

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
    });
    expect(key).toBeDefined();
    expect(key!.start.startsWith("prefix_")).toBe(true);
  });
});

describe("roles", () => {
  test("connects the specified roles", async (t) => {
    const h = await IntegrationHarness.init(t);
    const roles = ["r1", "r2"];
    await h.db.primary.insert(schema.roles).values(
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

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const key = await h.db.primary.query.keys.findFirst({
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
  test("upserts the specified roles", async (t) => {
    const h = await IntegrationHarness.init(t);
    const roles = ["r1", "r2"];

    const root = await h.createRootKey([
      `api.${h.resources.userApi.id}.create_key`,
      "rbac.*.create_role",
    ]);

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

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const key = await h.db.primary.query.keys.findFirst({
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

describe("permissions", () => {
  test("connects the specified permissions", async (t) => {
    const h = await IntegrationHarness.init(t);
    const permissions = ["p1", "p2"];
    await h.db.primary.insert(schema.permissions).values(
      permissions.map((name) => ({
        id: newId("test"),
        name,
        slug: name,
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
        permissions,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
      with: {
        permissions: {
          with: {
            permission: true,
          },
        },
      },
    });
    expect(key).toBeDefined();
    expect(key!.permissions.length).toBe(2);
    for (const p of key!.permissions!) {
      expect(permissions).include(p.permission.name);
    }
  });
});

test("upserts the specified permissions", async (t) => {
  const h = await IntegrationHarness.init(t);
  const permissions = ["p1", "p2"];

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.create_key`,
    "rbac.*.create_permission",
  ]);

  const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      permissions,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  let key = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.keyId),
    with: {
      permissions: {
        with: {
          permission: true,
        },
      },
    },
  });
  expect(key).toBeDefined();
  expect(key!.permissions.length).toBe(2);
  for (const p of key!.permissions!) {
    expect(permissions).include(p.permission.name);
  }

  const res2 = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      permissions,
    },
  });

  expect(res2.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  key = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.keyId),
    with: {
      permissions: {
        with: {
          permission: true,
        },
      },
    },
  });
  expect(key).toBeDefined();
  expect(key!.permissions.length).toBe(2);
  for (const p of key!.permissions!) {
    expect(permissions).include(p.permission.name);
  }
});

describe("with encryption", () => {
  test("encrypts a key", async (t) => {
    const h = await IntegrationHarness.init(t);

    await h.db.primary
      .update(schema.keyAuth)
      .set({
        storeEncryptedKeys: true,
      })
      .where(eq(schema.keyAuth.id, h.resources.userKeyAuth.id));

    const root = await h.createRootKey([
      `api.${h.resources.userApi.id}.create_key`,
      `api.${h.resources.userApi.id}.encrypt_key`,
    ]);

    const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        apiId: h.resources.userApi.id,
        recoverable: true,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
      with: {
        encrypted: true,
      },
    });
    expect(key).toBeDefined();
    expect(key!.encrypted).toBeDefined();
    expect(typeof key?.encrypted?.encrypted).toBe("string");
    expect(typeof key?.encrypted?.encryptionKeyId).toBe("string");
  });
});

test("creates a key with environment", async (t) => {
  const h = await IntegrationHarness.init(t);
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const key = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.keyId),
  });
  expect(key).toBeDefined();
  expect(key!.environment).toBe(environment);
});

describe("with ownerId", () => {
  describe("when ownerId does not exist yet", () => {
    test("should create identity", async (t) => {
      const h = await IntegrationHarness.init(t);

      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

      const ownerId = newId("test");
      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId: h.resources.userApi.id,
          ownerId,
        },
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const identity = await h.db.primary.query.identities.findFirst({
        where: (table, { eq }) => eq(table.externalId, ownerId),
        with: {
          keys: true,
        },
      });
      expect(identity).toBeDefined();

      const key = identity!.keys.at(0);
      expect(key).toBeDefined();
      expect(key!.id).toEqual(res.body.keyId);
    });
  });

  describe("when the identity exists already", () => {
    test("should link to the identity", async (t) => {
      const h = await IntegrationHarness.init(t);

      const externalId = newId("test");

      const identity = {
        id: newId("test"),
        externalId,
        workspaceId: h.resources.userWorkspace.id,
      };

      await h.db.primary.insert(schema.identities).values(identity);

      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId: h.resources.userApi.id,
          ownerId: externalId,
        },
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const key = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
        with: {
          identity: true,
        },
      });
      expect(key).toBeDefined();
      expect(key!.identity).toBeDefined();
      expect(key!.identity!.id).toEqual(identity.id);
    });
  });
});

describe("with externalId", () => {
  describe("when externalId does not exist yet", () => {
    test("should create identity", async (t) => {
      const h = await IntegrationHarness.init(t);

      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

      const externalId = newId("test");
      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId: h.resources.userApi.id,
          externalId,
        },
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const identity = await h.db.primary.query.identities.findFirst({
        where: (table, { eq }) => eq(table.externalId, externalId),
        with: {
          keys: true,
        },
      });
      expect(identity).toBeDefined();

      const key = identity!.keys.at(0);
      expect(key).toBeDefined();
      expect(key!.id).toEqual(res.body.keyId);
    });
  });

  describe("when the identity exists already", () => {
    test("should link to the identity", async (t) => {
      const h = await IntegrationHarness.init(t);

      const externalId = newId("test");

      const identity = {
        id: newId("test"),
        externalId,
        workspaceId: h.resources.userWorkspace.id,
      };

      await h.db.primary.insert(schema.identities).values(identity);

      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId: h.resources.userApi.id,
          ownerId: externalId,
        },
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const key = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
        with: {
          identity: true,
        },
      });
      expect(key).toBeDefined();
      expect(key!.identity).toBeDefined();
      expect(key!.identity!.id).toEqual(identity.id);
    });
  });
  describe("Should default first day of month if none provided", () => {
    test("should provide default value", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId: h.resources.userApi.id,
          remaining: 10,
          refill: {
            interval: "monthly",
            amount: 20,
            refillDay: undefined,
          },
        },
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const key = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyId),
        with: {
          credits: true,
        },
      });
      expect(key).toBeDefined();
      // Since key has remaining, it uses the new credits table
      expect(key!.credits).toBeDefined();
      expect(key!.credits!.refillDay).toEqual(1);
    });
  });
});

describe("creates key with new credits table", () => {
  test("creates credits table entry when remaining is specified", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

    const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        apiId: h.resources.userApi.id,
        remaining: 100,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
      with: {
        credits: true,
      },
    });

    expect(key).toBeDefined();
    expect(key!.remaining).toBeNull();

    // Verify credits table entry was created
    expect(key!.credits).toBeDefined();
    expect(key!.credits!.remaining).toEqual(100);
    expect(key!.credits!.keyId).toEqual(res.body.keyId);
    expect(key!.credits!.workspaceId).toEqual(h.resources.userWorkspace.id);
    expect(key!.credits!.identityId).toBeNull();
  });

  test("creates credits with refill configuration", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

    const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        apiId: h.resources.userApi.id,
        remaining: 50,
        refill: {
          interval: "monthly",
          amount: 100,
          refillDay: 15,
        },
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
      with: {
        credits: true,
      },
    });

    expect(key).toBeDefined();
    expect(key!.credits).toBeDefined();
    expect(key!.credits!.remaining).toEqual(50);
    expect(key!.credits!.refillAmount).toEqual(100);
    expect(key!.credits!.refillDay).toEqual(15);
    expect(key!.credits!.refilledAt).toBeDefined();
  });

  test("does not create credits table entry when remaining is not specified", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

    const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        apiId: h.resources.userApi.id,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
      with: {
        credits: true,
      },
    });

    expect(key).toBeDefined();
    expect(key!.remaining).toBeNull();
    expect(key!.credits).toBeNull();
  });

  test("creates key with remaining and can verify it", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

    const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        apiId: h.resources.userApi.id,
        remaining: 5,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    // Verify we can use the key
    const verifyRes = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key: res.body.key,
        apiId: h.resources.userApi.id,
      },
    });

    expect(verifyRes.status, `expected 200, received: ${JSON.stringify(verifyRes, null, 2)}`).toBe(
      200,
    );
    expect(verifyRes.body.valid).toBe(true);
    expect(verifyRes.body.remaining).toEqual(4);

    // Verify the credits table was updated
    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
      with: {
        credits: true,
      },
    });

    expect(key!.credits!.remaining).toEqual(4);
  });
});
