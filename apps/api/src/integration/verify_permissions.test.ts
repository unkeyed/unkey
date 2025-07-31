import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "@/routes/v1_keys_verifyKey";
import { describe, expect, test } from "vitest";

test("without permissions", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key } = await h.createKey();

  const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${h.baseUrl}/v1/keys.verifyKey`,
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      apiId: h.resources.userApi.id,
      key,
      authorization: {
        permissions: {
          and: ["p1", "p2"],
        },
      },
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.valid).toBe(false);
  expect(res.body.code).toBe("INSUFFICIENT_PERMISSIONS");
});

test("with roles but not permissions", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key } = await h.createKey({
    roles: [
      {
        name: "r1",
      },
    ],
  });

  const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${h.baseUrl}/v1/keys.verifyKey`,
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      apiId: h.resources.userApi.id,
      key,
      authorization: {
        permissions: {
          and: ["p1", "p2"],
        },
      },
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.valid).toBe(false);
  expect(res.body.code).toBe("INSUFFICIENT_PERMISSIONS");
});

test("with roles and insufficient permissions", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key } = await h.createKey({
    roles: [
      {
        name: "r1",
        permissions: ["p1", "p2"],
      },
    ],
  });

  const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${h.baseUrl}/v1/keys.verifyKey`,
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      apiId: h.resources.userApi.id,
      key,
      authorization: {
        permissions: "p3",
      },
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.valid).toBe(false);

  expect(res.body.code).toBe("INSUFFICIENT_PERMISSIONS");
});

test("has all required permissions", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key } = await h.createKey({
    roles: [
      {
        name: "r1",
        permissions: ["p1", "p2"],
      },
    ],
  });

  const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${h.baseUrl}/v1/keys.verifyKey`,
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      apiId: h.resources.userApi.id,
      key,
      authorization: {
        permissions: {
          and: ["p1", "p2"],
        },
      },
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.valid).toBe(true);
  expect(res.body.code).toBe("VALID");
});

describe(
  "many roles and permissions",
  () => {
    test("returns valid=true", async (t) => {
      const h = await IntegrationHarness.init(t);
      const { key } = await h.createKey({
        roles: [
          {
            name: "r1",
            permissions: ["p1", "p2", "p3"],
          },
          {
            name: "r2",
            permissions: ["p2", "p4", "p6"],
          },
          {
            name: "r3",
            permissions: ["p1", "p2", "p5"],
          },
          {
            name: "r4",
            permissions: ["p2", "p4", "p9"],
          },
          {
            name: "r5",
            permissions: ["p5", "p6", "p7"],
          },
          {
            name: "r6",
            permissions: [],
          },
          {
            name: "r7",
            permissions: ["p1", "p8", "p9", "p10"],
          },
          {
            name: "r8",
            permissions: ["p1", "p2", "p3"],
          },
        ],
      });

      const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
        url: `${h.baseUrl}/v1/keys.verifyKey`,
        headers: {
          "Content-Type": "application/json",
        },
        body: {
          apiId: h.resources.userApi.id,
          key,
          authorization: {
            permissions: {
              and: ["p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8", "p9", "p10"],
            },
          },
        },
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
      expect(res.body.valid).toBe(true);
      expect(res.body.permissions).toBeDefined();
      expect(res.body.permissions!.length).toBe(10);
      for (const p of ["p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8", "p9", "p10"]) {
        expect(res.body.permissions!).includes(p);
      }
    });
  },
  { timeout: 20_000 },
);

describe(
  "invalid permission query",
  () => {
    test("returns BAD_REQUEST", async (t) => {
      const h = await IntegrationHarness.init(t);
      const { key } = await h.createKey();

      const res = await h.post<V1KeysVerifyKeyRequest, { error: { code: string } }>({
        url: `${h.baseUrl}/v1/keys.verifyKey`,
        headers: {
          "Content-Type": "application/json",
        },
        body: {
          apiId: h.resources.userApi.id,
          key,
          authorization: {
            permissions: {
              and: ["p1", {}],
            },
          },
        },
      });

      expect(res.status).toBe(400);
      expect(res.body.error.code).toBe("BAD_REQUEST");
    });
  },
  { timeout: 20_000 },
);
