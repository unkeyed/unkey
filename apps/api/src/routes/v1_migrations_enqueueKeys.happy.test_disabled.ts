import { describe, expect, test, vi } from "vitest";

import { randomUUID } from "node:crypto";
import { eq, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { sha256 } from "@unkey/hash";
import { KeyV1 } from "@unkey/keys";
import type { V1KeysGetKeyResponse } from "./v1_keys_getKey";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "./v1_keys_verifyKey";
import type {
  V1MigrationsEnqueueKeysRequest,
  V1MigrationsEnqueueKeysResponse,
} from "./v1_migrations_enqueueKeys";

test("creates key", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const migrationId = newId("test");

  const hash = await sha256(randomUUID());
  const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
    url: "/v1/migrations.enqueueKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      migrationId,
      apiId: h.resources.userApi.id,
      keys: [
        {
          start: "start_",
          hash: {
            value: hash,
            variant: "sha256_base64",
          },
          enabled: true,
        },
      ],
    },
  });
  expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);

  await vi.waitFor(
    async () => {
      const found = await h.db.primary.query.keys.findMany({
        where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
      });
      expect(found.length).toBe(1);
      expect(found[0].hash).toEqual(hash);
    },
    {
      timeout: 20000,
    },
  );
}, 30000);

describe("with enabled flag", () => {
  describe("not set", () => {
    test("should still create an enabled key", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
      const hash = await sha256(randomUUID());
      const migrationId = newId("test");

      const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
        url: "/v1/migrations.enqueueKeys",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId: h.resources.userApi.id,
          migrationId,
          keys: [
            {
              start: "start_",
              hash: {
                value: hash,
                variant: "sha256_base64",
              },
            },
          ],
        },
      });

      expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);

      await vi.waitFor(
        async () => {
          const found = await h.db.primary.query.keys.findMany({
            where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
          });
          expect(found.length).toBe(1);
          expect(found[0].hash).toEqual(hash);
          expect(found[0].enabled).toBe(true);
        },
        {
          timeout: 20000,
        },
      );
    }, 30000);
  });
  describe("enabled: false", () => {
    test("should create a disabled key", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
      const migrationId = newId("test");

      const hash = await sha256(randomUUID());
      const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
        url: "/v1/migrations.enqueueKeys",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId: h.resources.userApi.id,
          migrationId,
          keys: [
            {
              start: "start_",
              hash: {
                value: hash,
                variant: "sha256_base64",
              },
              enabled: false,
            },
          ],
        },
      });

      expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);
      await vi.waitFor(
        async () => {
          const found = await h.db.primary.query.keys.findMany({
            where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
          });
          expect(found.length).toBe(1);
          expect(found[0].hash).toEqual(hash);
          expect(found[0].enabled).toBe(false);
        },
        { timeout: 20000, interval: 500 },
      );
    }, 30000);
  });
  describe("enabled: true", () => {
    test("should create an enabled key", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
      const migrationId = newId("test");

      const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
        url: "/v1/migrations.enqueueKeys",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId: h.resources.userApi.id,
          migrationId,
          keys: [
            {
              start: "start_",
              hash: {
                value: await sha256(randomUUID()),
                variant: "sha256_base64",
              },
              enabled: true,
            },
          ],
        },
      });

      expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);
      await vi.waitFor(
        async () => {
          const found = await h.db.primary.query.keys.findMany({
            where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
          });
          expect(found.length).toBe(1);
          expect(found[0].enabled).toBe(true);
        },
        {
          timeout: 20000,
        },
      );
    }, 30000);
  });
});

describe("with prefix", () => {
  test("start includes prefix", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
    const hash = await sha256(randomUUID());
    const migrationId = newId("test");

    const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
      url: "/v1/migrations.enqueueKeys",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        apiId: h.resources.userApi.id,
        migrationId,
        keys: [
          {
            start: "start_",
            hash: {
              value: hash,
              variant: "sha256_base64",
            },
            enabled: true,
          },
        ],
      },
    });

    expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);

    await vi.waitFor(
      async () => {
        const found = await h.db.primary.query.keys.findMany({
          where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
        });
        expect(found.length).toBe(1);
        expect(found[0].start).toEqual("start_");
      },
      { timeout: 20000, interval: 500 },
    );
  }, 30000);
});

describe("with metadata", () => {
  test("creates 100 keys", async (t) => {
    const h = await IntegrationHarness.init(t);
    const migrationId = newId("test");

    const root = await h.createRootKey([
      `api.${h.resources.userApi.id}.create_key`,
      `api.${h.resources.userApi.id}.encrypt_key`,
    ]);

    const keys = new Array(100).fill(null).map(
      (_, i) =>
        ({
          start: i.toString(),
          plaintext: crypto.randomUUID(),
          apiId: h.resources.userApi.id,
          meta: {
            a: Math.random(),
            b: Math.random(),
            c: Math.random(),
            d: Math.random(),
            e: Math.random(),
            f: Math.random(),
            g: Math.random(),
            h: Math.random(),
            i: Math.random(),
            j: Math.random(),
            k: Math.random(),
            l: Math.random(),
          },
        }) as const,
    );

    const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
      url: "/v1/migrations.enqueueKeys",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: { migrationId, apiId: h.resources.userApi.id, keys },
    });

    expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);

    for (let i = 0; i < keys.length; i++) {
      const hash = await sha256(keys[i].plaintext);
      await vi.waitFor(
        async () => {
          const key = await h.db.primary.query.keys.findFirst({
            where: (table, { eq }) => eq(table.hash, hash),
          });
          expect(key).toBeDefined();
          expect(JSON.parse(key!.meta!)).toMatchObject(keys[i].meta);
          expect(key!.start).toBe(keys[i].start);
        },
        { timeout: 20000, interval: 500 },
      );
    }
  }, 30000);
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
    const migrationId = newId("test");

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

    const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
      url: "/v1/migrations.enqueueKeys",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        apiId: h.resources.userApi.id,
        migrationId,
        keys: [
          {
            start: "start_",
            hash: {
              value: await sha256("hello world"),
              variant: "sha256_base64",
            },
            permissions,
          },
        ],
      },
    });

    expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);

    await vi.waitFor(
      async () => {
        const key = await h.db.primary.query.keys.findFirst({
          where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
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
      },
      { timeout: 20000, interval: 500 },
    );
  }, 30000);
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
    const migrationId = newId("test");

    const hash = await sha256(randomUUID());
    const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
      url: "/v1/migrations.enqueueKeys",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        apiId: h.resources.userApi.id,
        migrationId,
        keys: [
          {
            start: "start_",
            hash: {
              value: hash,
              variant: "sha256_base64",
            },
            roles,
          },
        ],
      },
    });

    expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);
    await vi.waitFor(
      async () => {
        const key = await h.db.primary.query.keys.findFirst({
          where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
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
      },
      { timeout: 20000, interval: 500 },
    );
  }, 30000);
});

test("creates a key with environment", async (t) => {
  const h = await IntegrationHarness.init(t);
  const environment = "test";
  const migrationId = newId("test");

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
  const hash = await sha256(randomUUID());

  const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
    url: "/v1/migrations.enqueueKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      migrationId,
      keys: [
        {
          start: "start_",
          hash: {
            value: hash,
            variant: "sha256_base64",
          },
          environment,
        },
      ],
    },
  });

  expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);
  await vi.waitFor(
    async () => {
      const key = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
      });
      expect(key).toBeDefined();
      expect(key!.environment).toBe(environment);
    },
    { timeout: 20000, interval: 500 },
  );
}, 30000);

test("creates and encrypts a key", async (t) => {
  const h = await IntegrationHarness.init(t);
  const plaintext = crypto.randomUUID();
  const migrationId = newId("test");

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.create_key`,
    "api.*.encrypt_key",
  ]);

  const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
    url: "/v1/migrations.enqueueKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      migrationId,
      keys: [
        {
          start: plaintext.slice(0, 5),
          plaintext,
        },
      ],
    },
  });

  expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);
  await vi.waitFor(
    async () => {
      const key = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
      });
      expect(key).toBeDefined();
      expect(key!.hash).toBe(await sha256(plaintext));
    },
    { timeout: 20000, interval: 500 },
  );
}, 30000);

test("creates a key with ratelimit", async (t) => {
  const h = await IntegrationHarness.init(t);
  const migrationId = newId("test");

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
  const hash = await sha256(randomUUID());
  const ratelimit = {
    async: false,
    limit: 999,
    duration: 1000,
  };

  const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
    url: "/v1/migrations.enqueueKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      migrationId,
      keys: [
        {
          start: "start_",
          hash: {
            value: hash,
            variant: "sha256_base64",
          },
          ratelimit,
        },
      ],
    },
  });

  expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);
  await vi.waitFor(
    async () => {
      const key = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
        with: {
          ratelimits: true,
        },
      });
      expect(key).toBeDefined();
      expect(key!.ratelimits.length).toBe(1);
      expect(key!.ratelimits[0].limit).toBe(ratelimit.limit);
      expect(key!.ratelimits[0].duration).toBe(ratelimit.duration);
    },
    { timeout: 20000, interval: 500 },
  );
}, 30000);

test("creates a key with remaining", async (t) => {
  const h = await IntegrationHarness.init(t);
  const migrationId = newId("test");

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
  const hash = await sha256(randomUUID());
  const remaining = 999;

  const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
    url: "/v1/migrations.enqueueKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      migrationId,
      keys: [
        {
          start: "start_",
          hash: {
            value: hash,
            variant: "sha256_base64",
          },
          remaining,
        },
      ],
    },
  });

  expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);
  await vi.waitFor(
    async () => {
      const key = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
      });
      expect(key).toBeDefined();
      expect(key!.remaining).toBe(remaining);
    },
    { timeout: 20000, interval: 500 },
  );
}, 30000);

test("creates 100 keys", async (t) => {
  const h = await IntegrationHarness.init(t);
  const migrationId = newId("test");

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.create_key`,
    `api.${h.resources.userApi.id}.encrypt_key`,
  ]);

  const keys = new Array(100).fill(null).map(
    (_, i) =>
      ({
        start: i.toString(),
        plaintext: crypto.randomUUID(),
        apiId: h.resources.userApi.id,
        enabled: Math.random() > 0.5,
      }) as const,
  );

  const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
    url: "/v1/migrations.enqueueKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: { migrationId, apiId: h.resources.userApi.id, keys },
  });

  expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);

  for (let i = 0; i < keys.length; i++) {
    await vi.waitFor(
      async () => {
        const hash = await sha256(keys[i].plaintext);
        const key = await h.db.primary.query.keys.findFirst({
          where: (table, { eq }) => eq(table.hash, hash),
        });
        expect(key).toBeDefined();
        expect(key!.enabled).toBe(keys[i].enabled);
        expect(key!.start).toBe(keys[i].start);
      },
      { timeout: 45000 },
    );
  }
}, 60000);

test("retrieves a key in plain text", async (t) => {
  const h = await IntegrationHarness.init(t);
  const migrationId = newId("test");

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.create_key`,
    `api.${h.resources.userApi.id}.read_key`,
    `api.${h.resources.userApi.id}.decrypt_key`,
    `api.${h.resources.userApi.id}.encrypt_key`,
  ]);

  await h.db.primary
    .update(schema.keyAuth)
    .set({
      storeEncryptedKeys: true,
    })
    .where(eq(schema.keyAuth.id, h.resources.userKeyAuth.id));

  const key = new KeyV1({ byteLength: 16, prefix: "test" }).toString();
  const hash = await sha256(key);

  const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
    url: "/v1/migrations.enqueueKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      migrationId,
      keys: [
        {
          plaintext: key,
        },
      ],
    },
  });
  expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);
  const found = await vi.waitFor(
    async () => {
      const f = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.keyAuthId, h.resources.userKeyAuth.id),
      });

      expect(f!.hash).toEqual(hash);
      return f;
    },
    { timeout: 20000, interval: 500 },
  );

  const getKeyRes = await h.get<V1KeysGetKeyResponse>({
    url: `/v1/keys.getKey?keyId=${found!.id}&decrypt=true`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(getKeyRes.status).toBe(200);
  expect(getKeyRes.body.plaintext).toEqual(key);
}, 20000);

test("migrate and verify a key", async (t) => {
  const h = await IntegrationHarness.init(t);

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.create_key`,
    "api.*.encrypt_key",
  ]);
  const migrationId = newId("test");

  const key = new KeyV1({ byteLength: 16, prefix: "test" }).toString();
  const hash = await sha256(key);

  const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
    url: "/v1/migrations.enqueueKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      migrationId,
      keys: [
        {
          plaintext: key,
        },
      ],
    },
  });
  expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);

  await vi.waitFor(
    async () => {
      const f = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.hash, hash),
      });

      expect(f).toBeDefined();
    },
    { timeout: 20000, interval: 500 },
  );

  await vi.waitFor(
    async () => {
      const verifyRes = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
        url: "/v1/keys.verifyKey",
        headers: {
          "Content-Type": "application/json",
        },
        body: {
          apiId: h.resources.userApi.id,
          key,
        },
      });
      expect(verifyRes.status, `expected 200, received: ${JSON.stringify(verifyRes)}`).toBe(200);
      expect(verifyRes.body.valid, JSON.stringify(verifyRes)).toEqual(true);
    },
    { timeout: 20000, interval: 500 },
  );
}, 30000);
