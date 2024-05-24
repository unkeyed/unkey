import { describe, expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { sha256 } from "@unkey/hash";
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
  expect(res.status).toEqual(200);

  const found = await h.db.readonly.query.keys.findFirst({
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

      expect(res.status).toEqual(200);

      const found = await h.db.readonly.query.keys.findFirst({
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

      expect(res.status).toEqual(200);

      const found = await h.db.readonly.query.keys.findFirst({
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

      expect(res.status).toEqual(200);

      const found = await h.db.readonly.query.keys.findFirst({
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

    expect(res.status).toEqual(200);

    const key = await h.db.readonly.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
    });
    expect(key).toBeDefined();
    expect(key!.start).toEqual("start_");
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

    expect(res.status).toEqual(200);

    const key = await h.db.readonly.query.keys.findFirst({
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

  expect(res.status).toEqual(200);

  const key = await h.db.readonly.query.keys.findFirst({
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

  expect(res.status).toEqual(200);
  expect(res.body.keyIds.length).toEqual(req.length);

  for (let i = 0; i < req.length; i++) {
    const key = await h.db.readonly.query.keys.findFirst({
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

  const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
    url: "/v1/migrations.createKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: req,
  });

  expect(res.status).toEqual(500);

  for (let i = 0; i < req.length; i++) {
    const key = await h.db.readonly.query.keys.findFirst({
      where: (table, { eq }) => eq(table.hash, req[i].hash.value),
    });
    expect(key).toBeUndefined();
  }
});
