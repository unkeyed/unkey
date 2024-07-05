import { describe, expect, test } from "vitest";

import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { ErrorResponse } from "@unkey/api/src";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "./v1_keys_verifyKey";

describe("without identities", () => {
  test("returns valid", async (t) => {
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
        ratelimits: [
          {
            name: "10/10s",
            cost: 4,
            limit: 10,
            duration: 10_000,
          },
          {
            name: "1/1min",
            cost: 1,
            limit: 1,
            duration: 60_000,
          },
        ],
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res)}`).toBe(200);
    expect(res.body.valid).toBe(true);
    expect(res.body.code).toBe("VALID");
  });

  test("returns RATE_LIMITED", async (t) => {
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
        ratelimits: [
          {
            name: "10/10s",
            cost: 4,
            limit: 10,
            duration: 10_000,
          },
          {
            name: "1/1min",
            cost: 2,
            limit: 1,
            duration: 60_000,
          },
        ],
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res)}`).toBe(200);
    expect(res.body.valid).toBe(false);
    expect(res.body.code).toBe("RATE_LIMITED");
  });

  describe("without config", () => {
    test("returns error", async (t) => {
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

      const res = await h.post<V1KeysVerifyKeyRequest, ErrorResponse>({
        url: "/v1/keys.verifyKey",
        headers: {
          "Content-Type": "application/json",
        },
        body: {
          key,
          apiId: h.resources.userApi.id,
          ratelimits: [
            {
              name: "missing_config",
              cost: 4,
            },
            {
              name: "1/1min",
              cost: 2,
              limit: 1,
              duration: 60_000,
            },
          ],
        },
      });

      expect(res.status, `expected 400, received: ${JSON.stringify(res)}`).toBe(400);
    });
  });
});

describe("with identity", () => {
  describe("with ratelimits on key and identity", () => {
    test("ratelimits on key should take precedence and pass", async (t) => {
      const h = await IntegrationHarness.init(t);

      const identityId = newId("test");
      await h.db.primary.insert(schema.identities).values({
        id: identityId,
        externalId: "test",
        environment: "test",
        workspaceId: h.resources.userWorkspace.id,
      });

      const keyId = newId("test");
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: keyId,
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        identityId,
      });

      const keyLimit1 = {
        id: newId("test"),
        keyId,
        limit: 4,
        duration: 10_000,
        name: "limit1",
        workspaceId: h.resources.userWorkspace.id,
      };
      const idLimit1 = {
        id: newId("test"),
        identityId,
        limit: 1,
        duration: 600_000,
        name: "limit1",
        workspaceId: h.resources.userWorkspace.id,
      };
      const idLimit2 = {
        id: newId("test"),
        identityId,
        limit: 10,
        duration: 600_000,
        name: "limit2",
        workspaceId: h.resources.userWorkspace.id,
      };

      await h.db.primary.insert(schema.ratelimits).values([keyLimit1, idLimit1, idLimit2]);

      const testCases: TestCase[] = [
        {
          ratelimits: [{ name: "limit1" }, { name: "limit2" }],
          expected: {
            status: 200,
            valid: true,
            code: "VALID",
          },
        },
        {
          ratelimits: [{ name: "limit1" }, { name: "limit2" }],
          expected: {
            status: 200,
            valid: true,
            code: "VALID",
          },
        },
        {
          ratelimits: [{ name: "limit1" }, { name: "limit2" }],
          expected: {
            status: 200,
            valid: true,
            code: "VALID",
          },
        },
      ];

      for (const tc of testCases) {
        const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
          url: "/v1/keys.verifyKey",
          headers: {
            "Content-Type": "application/json",
          },
          body: {
            key,
            apiId: h.resources.userApi.id,
            ratelimits: tc.ratelimits,
          },
        });

        expect(res.status, `received: ${JSON.stringify(res, null, 2)}`).toBe(tc.expected.status);
        expect(res.body.valid).toBe(tc.expected.valid);
        expect(res.body.code).toBe(tc.expected.code);
        expect(res.headers["unkey-ratelimit-triggered"]).toEqual(tc.expected.triggered);
      }
    });
    test("ratelimits on key should take precedence and reject", async (t) => {
      const h = await IntegrationHarness.init(t);

      const identityId = newId("test");
      await h.db.primary.insert(schema.identities).values({
        id: identityId,
        externalId: "test",
        environment: "test",
        workspaceId: h.resources.userWorkspace.id,
      });

      const keyId = newId("test");
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: keyId,
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        identityId,
      });

      const keyLimit1 = {
        id: newId("test"),
        keyId,
        limit: 1,
        duration: 10_000,
        name: "limit1",
        workspaceId: h.resources.userWorkspace.id,
      };
      const idLimit1 = {
        id: newId("test"),
        identityId,
        limit: 10,
        duration: 600_000,
        name: "limit1",
        workspaceId: h.resources.userWorkspace.id,
      };
      const idLimit2 = {
        id: newId("test"),
        identityId,
        limit: 10,
        duration: 600_000,
        name: "limit2",
        workspaceId: h.resources.userWorkspace.id,
      };

      await h.db.primary.insert(schema.ratelimits).values([keyLimit1, idLimit1, idLimit2]);

      const testCases: TestCase[] = [
        {
          ratelimits: [{ name: "limit1" }, { name: "limit2" }],
          expected: {
            status: 200,
            valid: true,
            code: "VALID",
          },
        },
        {
          ratelimits: [{ name: "limit1" }, { name: "limit2" }],
          expected: {
            status: 200,
            valid: false,
            code: "RATE_LIMITED",
            triggered: "limit1",
          },
        },
        {
          ratelimits: [{ name: "limit1" }, { name: "limit2" }],
          expected: {
            status: 200,
            valid: false,
            code: "RATE_LIMITED",
            triggered: "limit1",
          },
        },
      ];

      for (const tc of testCases) {
        const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
          url: "/v1/keys.verifyKey",
          headers: {
            "Content-Type": "application/json",
          },
          body: {
            key,
            apiId: h.resources.userApi.id,
            ratelimits: tc.ratelimits,
          },
        });

        expect(res.status, `received: ${JSON.stringify(res, null, 2)}`).toBe(tc.expected.status);
        expect(res.body.valid).toBe(tc.expected.valid);
        expect(res.body.code).toBe(tc.expected.code);
        expect(res.headers["unkey-ratelimit-triggered"]).toEqual(tc.expected.triggered);
      }
    });
    test("fallback limits must still reject", async (t) => {
      const h = await IntegrationHarness.init(t);

      const identityId = newId("test");
      await h.db.primary.insert(schema.identities).values({
        id: identityId,
        externalId: "test",
        environment: "test",
        workspaceId: h.resources.userWorkspace.id,
      });

      const keyId = newId("test");
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: keyId,
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        identityId,
      });

      const keyLimit1 = {
        id: newId("test"),
        keyId,
        limit: 4,
        duration: 10_000,
        name: "limit1",
        workspaceId: h.resources.userWorkspace.id,
      };
      const idLimit1 = {
        id: newId("test"),
        identityId,
        limit: 10,
        duration: 600_000,
        name: "limit1",
        workspaceId: h.resources.userWorkspace.id,
      };
      const idLimit2 = {
        id: newId("test"),
        identityId,
        limit: 2,
        duration: 600_000,
        name: "limit2",
        workspaceId: h.resources.userWorkspace.id,
      };

      await h.db.primary.insert(schema.ratelimits).values([keyLimit1, idLimit1, idLimit2]);

      const testCases: TestCase[] = [
        {
          ratelimits: [{ name: "limit1" }, { name: "limit2" }],
          expected: {
            /* The above code appears to be a TypeScript object with three properties: status, valid, and code. The status property is set to 200, the valid property is set to true, and the code property is set to "VALID". */
            status: 200,
            valid: true,
            code: "VALID",
          },
        },
        {
          ratelimits: [{ name: "limit1" }, { name: "limit2" }],
          expected: {
            status: 200,
            valid: true,
            code: "VALID",
          },
        },
        {
          ratelimits: [{ name: "limit1" }, { name: "limit2" }],
          expected: {
            status: 200,
            valid: false,
            code: "RATE_LIMITED",
            triggered: "limit2",
          },
        },
      ];

      for (const tc of testCases) {
        const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
          url: "/v1/keys.verifyKey",
          headers: {
            "Content-Type": "application/json",
          },
          body: {
            key,
            apiId: h.resources.userApi.id,
            ratelimits: tc.ratelimits,
          },
        });

        expect(res.status, `received: ${JSON.stringify(res, null, 2)}`).toBe(tc.expected.status);
        expect(res.body.valid).toBe(tc.expected.valid);
        expect(res.body.code).toBe(tc.expected.code);
        expect(res.headers["unkey-ratelimit-triggered"]).toEqual(tc.expected.triggered);
      }
    });
    test("both limits must reject", async (t) => {
      const h = await IntegrationHarness.init(t);

      const identityId = newId("test");
      await h.db.primary.insert(schema.identities).values({
        id: identityId,
        externalId: "test",
        environment: "test",
        workspaceId: h.resources.userWorkspace.id,
      });

      const keyId = newId("test");
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: keyId,
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        identityId,
      });

      const keyLimit1 = {
        id: newId("test"),
        keyId,
        limit: 1,
        duration: 10_000,
        name: "limit1",
        workspaceId: h.resources.userWorkspace.id,
      };
      const idLimit1 = {
        id: newId("test"),
        identityId,
        limit: 10,
        duration: 600_000,
        name: "limit1",
        workspaceId: h.resources.userWorkspace.id,
      };
      const idLimit2 = {
        id: newId("test"),
        identityId,
        limit: 2,
        duration: 600_000,
        name: "limit2",
        workspaceId: h.resources.userWorkspace.id,
      };

      await h.db.primary.insert(schema.ratelimits).values([keyLimit1, idLimit1, idLimit2]);

      const testCases: TestCase[] = [
        {
          ratelimits: [{ name: "limit1" }, { name: "limit2" }],
          expected: {
            /* The above code appears to be a TypeScript object with three properties: status, valid, and code. The status property is set to 200, the valid property is set to true, and the code property is set to "VALID". */
            status: 200,
            valid: true,
            code: "VALID",
          },
        },
        {
          ratelimits: [{ name: "limit1" }, { name: "limit2" }],
          expected: {
            status: 200,
            valid: false,
            code: "RATE_LIMITED",
            triggered: "limit1",
          },
        },
        {
          ratelimits: [{ name: "limit2" }],
          expected: {
            status: 200,
            valid: false,
            code: "RATE_LIMITED",
            triggered: "limit2",
          },
        },
      ];

      for (const tc of testCases) {
        const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
          url: "/v1/keys.verifyKey",
          headers: {
            "Content-Type": "application/json",
          },
          body: {
            key,
            apiId: h.resources.userApi.id,
            ratelimits: tc.ratelimits,
          },
        });

        expect(res.status, `received: ${JSON.stringify(res, null, 2)}`).toBe(tc.expected.status);
        expect(res.body.valid).toBe(tc.expected.valid);
        expect(res.body.code).toBe(tc.expected.code);
        expect(res.headers["unkey-ratelimit-triggered"]).toEqual(tc.expected.triggered);
      }
    });
  });

  describe("without specifying ratelimits in request", () => {
    test("should use the 'default' limit", async (t) => {
      const h = await IntegrationHarness.init(t);

      const identityId = newId("test");
      await h.db.primary.insert(schema.identities).values({
        id: identityId,
        externalId: "test",
        environment: "test",
        workspaceId: h.resources.userWorkspace.id,
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: newId("test"),
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        identityId,
        ratelimitAsync: true,
        ratelimitLimit: 1,
        ratelimitDuration: 10_000,
      });

      const testCases: TestCase[] = [
        {
          expected: {
            status: 200,
            valid: true,
            code: "VALID",
          },
        },
        {
          expected: {
            status: 200,
            valid: false,
            code: "RATE_LIMITED",
            triggered: "default",
          },
        },
        {
          expected: {
            status: 200,
            valid: false,
            code: "RATE_LIMITED",
            triggered: "default",
          },
        },
      ];

      for (const tc of testCases) {
        const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
          url: "/v1/keys.verifyKey",
          headers: {
            "Content-Type": "application/json",
          },
          body: {
            key,
            apiId: h.resources.userApi.id,
            ratelimits: tc.ratelimits,
          },
        });

        expect(res.status, `received: ${JSON.stringify(res, null, 2)}`).toBe(tc.expected.status);
        expect(res.body.valid).toBe(tc.expected.valid);
        expect(res.body.code).toBe(tc.expected.code);
        expect(res.headers["unkey-ratelimit-triggered"]).toEqual(tc.expected.triggered);
      }
    });
  });

  describe("falls back to limits defined for the identity", () => {
    test("should reject after the first limit hit", async (t) => {
      const h = await IntegrationHarness.init(t);

      const identityId = newId("test");
      await h.db.primary.insert(schema.identities).values({
        id: identityId,
        externalId: "test",
        environment: "test",
        workspaceId: h.resources.userWorkspace.id,
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: newId("test"),
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        identityId,
      });

      const tokenLimit = {
        id: newId("test"),
        identityId,
        limit: 10,
        duration: 10_000,
        name: "tokens",
        workspaceId: h.resources.userWorkspace.id,
      };
      const requestLimit = {
        id: newId("test"),
        identityId,
        limit: 10,
        duration: 600_000,
        name: "10_per_10m",
        workspaceId: h.resources.userWorkspace.id,
      };

      await h.db.primary.insert(schema.ratelimits).values([tokenLimit, requestLimit]);

      const testCases: TestCase[] = [
        {
          ratelimits: [
            {
              name: tokenLimit.name,
              cost: 4,
            },
            {
              name: requestLimit.name,
            },
          ],
          expected: {
            status: 200,
            valid: true,
            code: "VALID",
          },
        },
        {
          ratelimits: [
            {
              name: tokenLimit.name,
              cost: 6,
            },
            {
              name: requestLimit.name,
            },
          ],
          expected: {
            status: 200,
            valid: true,
            code: "VALID",
          },
        },
        {
          ratelimits: [
            {
              name: tokenLimit.name,
              cost: 1,
            },
            {
              name: requestLimit.name,
            },
          ],
          expected: {
            status: 200,
            valid: false,
            code: "RATE_LIMITED",
            triggered: tokenLimit.name,
          },
        },
      ];

      for (const tc of testCases) {
        const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
          url: "/v1/keys.verifyKey",
          headers: {
            "Content-Type": "application/json",
          },
          body: {
            key,
            apiId: h.resources.userApi.id,
            ratelimits: tc.ratelimits,
          },
        });

        expect(res.status, `received: ${JSON.stringify(res, null, 2)}`).toBe(tc.expected.status);
        expect(res.body.valid).toBe(tc.expected.valid);
        expect(res.body.code).toBe(tc.expected.code);
        expect(res.headers["unkey-ratelimit-triggered"]).toEqual(tc.expected.triggered);
      }
    });
  });
});

type TestCase = {
  ratelimits?: V1KeysVerifyKeyRequest["ratelimits"];
  expected: {
    status: number;
    valid: boolean;
    code: V1KeysVerifyKeyResponse["code"];
    /**
     * The name of the ratelimit that should have been triggered
     */
    triggered?: string;
  };
};
