import { describe, expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { eq, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { ErrorResponse } from "@unkey/api/src";
import { sha256 } from "@unkey/hash";
import { KeyV1 } from "@unkey/keys";
import type { V1KeysGetKeyResponse } from "./v1_keys_getKey";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "./v1_keys_verifyKey";
import type {
  V1MigrationsCreateKeysRequest,
  V1MigrationsCreateKeysResponse,
} from "./v1_migrations_createKey";

test("creates key", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const hash = await sha256(randomUUID());
  const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
    url: "/v1/migrations.createKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: [
      {
        start: "start_",
        hash: {
          value: hash,
          variant: "sha256_base64",
        },
        apiId: h.resources.userApi.id,
        enabled: true,
      },
    ],
  });
  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
  });
  expect(found).toBeDefined();
  expect(found!.hash).toEqual(hash);
});

describe("with enabled flag", () => {
  describe("not set", () => {
    test("should still create an enabled key", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
      const hash = await sha256(randomUUID());

      const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
        url: "/v1/migrations.createKeys",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: [
          {
            start: "start_",
            hash: {
              value: hash,
              variant: "sha256_base64",
            },
            apiId: h.resources.userApi.id,
          },
        ],
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const found = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
      });
      expect(found).toBeDefined();
      expect(found!.hash).toEqual(hash);
      expect(found!.enabled).toBe(true);
    });
  });
  describe("enabled: false", () => {
    test("should create a disabled key", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

      const hash = await sha256(randomUUID());
      const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
        url: "/v1/migrations.createKeys",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: [
          {
            start: "start_",
            hash: {
              value: hash,
              variant: "sha256_base64",
            },
            apiId: h.resources.userApi.id,
            enabled: false,
          },
        ],
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const found = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
      });
      expect(found).toBeDefined();
      expect(found!.hash).toEqual(hash);
      expect(found!.enabled).toBe(false);
    });
  });
  describe("enabled: true", () => {
    test("should create an enabled key", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

      const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
        url: "/v1/migrations.createKeys",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: [
          {
            start: "start_",
            hash: {
              value: await sha256(randomUUID()),
              variant: "sha256_base64",
            },
            apiId: h.resources.userApi.id,
            enabled: true,
          },
        ],
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const found = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
      });
      expect(found).toBeDefined();
      expect(found!.enabled).toBe(true);
    });
  });
});

describe("with prefix", () => {
  test("start includes prefix", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
    const hash = await sha256(randomUUID());

    const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
      url: "/v1/migrations.createKeys",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: [
        {
          start: "start_",
          hash: {
            value: hash,
            variant: "sha256_base64",
          },
          apiId: h.resources.userApi.id,
          enabled: true,
        },
      ],
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
    });
    expect(key).toBeDefined();
    expect(key!.start).toEqual("start_");
  });
});

describe("with externalId", () => {
  describe("when identity does not exist", () => {
    test("should create an identity", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
      const hash = await sha256(randomUUID());

      const externalId = newId("test");

      const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
        url: "/v1/migrations.createKeys",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: [
          {
            hash: {
              value: hash,
              variant: "sha256_base64",
            },
            apiId: h.resources.userApi.id,
            enabled: true,
            externalId: externalId,
          },
        ],
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const key = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
        with: {
          identity: true,
        },
      });
      expect(key).toBeDefined();

      expect(key!.identity).toBeDefined();
      expect(key!.identity!.externalId).toEqual(externalId);
    });
  });
  describe("when identity does exist", () => {
    test("should link the identity", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
      const hash = await sha256(randomUUID());

      const externalId = newId("test");
      const identity = {
        id: newId("test"),
        workspaceId: h.resources.userWorkspace.id,
        externalId,
      };
      await h.db.primary.insert(schema.identities).values(identity);

      const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
        url: "/v1/migrations.createKeys",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: [
          {
            hash: {
              value: hash,
              variant: "sha256_base64",
            },
            apiId: h.resources.userApi.id,
            enabled: true,
            externalId: externalId,
          },
        ],
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const key = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
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

describe("roles", () => {
  test("connects the specified roles", async (t) => {
    const h = await IntegrationHarness.init(t);
    const permissions = ["p1", "p2"];
    await h.db.primary.insert(schema.permissions).values(
      permissions.map((name) => ({
        id: newId("test"),
        name,
        workspaceId: h.resources.userWorkspace.id,
      })),
    );

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

    const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
      url: "/v1/migrations.createKeys",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: [
        {
          start: "start_",
          plaintext: "plaintext",
          apiId: h.resources.userApi.id,
          permissions,
        },
      ],
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
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

describe("permissions", () => {
  test("connects the specified permissions", async (t) => {
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

    const hash = await sha256(randomUUID());
    const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
      url: "/v1/migrations.createKeys",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: [
        {
          start: "start_",
          hash: {
            value: hash,
            variant: "sha256_base64",
          },
          apiId: h.resources.userApi.id,
          roles,
        },
      ],
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
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

test("creates a key with environment", async (t) => {
  const h = await IntegrationHarness.init(t);
  const environment = "test";

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);
  const hash = await sha256(randomUUID());

  const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
    url: "/v1/migrations.createKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: [
      {
        start: "start_",
        hash: {
          value: hash,
          variant: "sha256_base64",
        },
        apiId: h.resources.userApi.id,
        environment,
      },
    ],
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const key = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
  });
  expect(key).toBeDefined();
  expect(key!.environment).toBe(environment);
});

test("creates 100 keys", async (t) => {
  const h = await IntegrationHarness.init(t);

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const req = new Array(100).fill(null).map(
    (_, i) =>
      ({
        start: i.toString(),
        hash: {
          value: randomUUID(),
          variant: "sha256_base64",
        },
        apiId: h.resources.userApi.id,
        enabled: Math.random() > 0.5,
      }) as const,
  );

  const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
    url: "/v1/migrations.createKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: req,
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.keyIds.length).toEqual(req.length);

  for (let i = 0; i < req.length; i++) {
    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.hash, req[i].hash.value),
    });
    expect(key).toBeDefined();
    expect(key!.id).toBe(res.body.keyIds[i]);
    expect(key!.enabled).toBe(req[i].enabled);
    expect(key!.start).toBe(req[i].start);
  }
});

test("an error rolls back and does not create any keys", async (t) => {
  const h = await IntegrationHarness.init(t);

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const req = new Array(10).fill(null).map(
    (_, i) =>
      ({
        start: i.toString(),
        hash: {
          value: randomUUID(),
          variant: "sha256_base64",
        },
        apiId: h.resources.userApi.id,
        enabled: Math.random() > 0.5,
      }) as const,
  );
  // add a duplicate
  req.push(req[0]);

  const res = await h.post<V1MigrationsCreateKeysRequest, ErrorResponse>({
    url: "/v1/migrations.createKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: req,
  });

  expect(res.status).toEqual(409);
  expect(res.body.error.code).toEqual("NOT_UNIQUE");

  for (let i = 0; i < req.length; i++) {
    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.hash, req[i].hash.value),
    });
    expect(key).toBeUndefined();
  }
});

test("retrieves a key in plain text", async (t) => {
  const h = await IntegrationHarness.init(t);

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.create_key`,
    `api.${h.resources.userApi.id}.read_key`,
    `api.${h.resources.userApi.id}.decrypt_key`,
  ]);

  await h.db.primary
    .update(schema.keyAuth)
    .set({
      storeEncryptedKeys: true,
    })
    .where(eq(schema.keyAuth.id, h.resources.userKeyAuth.id));

  const key = new KeyV1({ byteLength: 16, prefix: "test" }).toString();
  const hash = await sha256(key);

  const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
    url: "/v1/migrations.createKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: [
      {
        plaintext: key,
        apiId: h.resources.userApi.id,
      },
    ],
  });
  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.keyIds.length).toEqual(1);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
  });

  expect(found!.hash).toEqual(hash);

  const getKeyRes = await h.get<V1KeysGetKeyResponse>({
    url: `/v1/keys.getKey?keyId=${res.body.keyIds[0]}&decrypt=true`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(getKeyRes.status).toBe(200);
  expect(getKeyRes.body.plaintext).toEqual(key);
});

test("migrate and verify a key", async (t) => {
  const h = await IntegrationHarness.init(t);

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const key = new KeyV1({ byteLength: 16, prefix: "test" }).toString();

  const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
    url: "/v1/migrations.createKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: [
      {
        plaintext: key,
        apiId: h.resources.userApi.id,
      },
    ],
  });
  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.keyIds.length).toEqual(1);

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

  expect(verifyRes.status).toBe(200);
  expect(verifyRes.body.valid).toEqual(true);
});

describe("Should default to first day of month if none provided", () => {
  test("should provide default value", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

    const hash = await sha256(randomUUID());
    const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
      url: "/v1/migrations.createKeys",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: [
        {
          start: "start_",
          hash: {
            value: hash,
            variant: "sha256_base64",
          },
          apiId: h.resources.userApi.id,
          enabled: true,
          remaining: 10,
          refill: {
            interval: "monthly",
            amount: 100,
            refillDay: undefined,
          },
        },
      ],
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const found = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
    });
    expect(found).toBeDefined();
    expect(found?.remaining).toEqual(10);
    expect(found?.refillAmount).toEqual(100);
    expect(found?.refillDay).toEqual(1);
    expect(found?.hash).toEqual(hash);
  });
});
