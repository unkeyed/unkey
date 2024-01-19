import { describe, expect, test } from "vitest";

import { Harness } from "@/pkg/testutil/harness";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import {
  type LegacyApisListKeysResponse,
  registerLegacyApisListKeys,
} from "./legacy_apis_listKeys";

describe("simple", () => {
  test("returns 200", async () => {
    const h = await Harness.init();
    h.useRoutes(registerLegacyApisListKeys);

    const keyIds = new Array(10).fill(0).map(() => newId("key"));
    for (let i = 0; i < keyIds.length; i++) {
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.resources.database.insert(schema.keys).values({
        id: keyIds[i],
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
      });
    }

    const res = await h.get<LegacyApisListKeysResponse>({
      url: `/v1/apis/${h.resources.userApi.id}/keys`,
      headers: {
        Authorization: `Bearer ${h.resources.rootKey}`,
      },
    });

    expect(res.status).toEqual(200);
    expect(res.body.total).toBeGreaterThanOrEqual(keyIds.length);
    expect(res.body.keys.length).toBeGreaterThanOrEqual(keyIds.length);
    expect(res.body.keys.length).toBeLessThanOrEqual(100); //  default page size
  });
});

describe("filter by ownerId", () => {
  test("returns all keys owned ", async () => {
    const h = await Harness.init();
    h.useRoutes(registerLegacyApisListKeys);

    const ownerId = crypto.randomUUID();
    const keyIds = new Array(10).fill(0).map(() => newId("key"));
    for (let i = 0; i < keyIds.length; i++) {
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.resources.database.insert(schema.keys).values({
        id: keyIds[i],
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        ownerId: i % 2 === 0 ? ownerId : undefined,
      });
    }

    const res = await h.get<LegacyApisListKeysResponse>({
      url: `/v1/apis/${h.resources.userApi.id}/keys?ownerId=${ownerId}`,
      headers: {
        Authorization: `Bearer ${h.resources.rootKey}`,
      },
    });

    expect(res.status).toEqual(200);
    expect(res.body.total).toBeGreaterThanOrEqual(keyIds.length);
    expect(res.body.keys).toHaveLength(5);
  });
});

describe("with limit", () => {
  test("returns only a few keys", async () => {
    const h = await Harness.init();
    h.useRoutes(registerLegacyApisListKeys);

    const keyIds = new Array(10).fill(0).map(() => newId("key"));
    for (let i = 0; i < keyIds.length; i++) {
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.resources.database.insert(schema.keys).values({
        id: keyIds[i],
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
      });
    }

    const res = await h.get<LegacyApisListKeysResponse>({
      url: `/v1/apis/${h.resources.userApi.id}/keys?limit=2`,
      headers: {
        Authorization: `Bearer ${h.resources.rootKey}`,
      },
    });
    expect(res.status).toEqual(200);
    expect(res.body.total).toBeGreaterThanOrEqual(keyIds.length);
    expect(res.body.keys).toHaveLength(2);
  }, 10_000);
});

describe("with offset", () => {
  test("returns the correct keys", async () => {
    const h = await Harness.init();
    h.useRoutes(registerLegacyApisListKeys);

    const keyIds = new Array(10).fill(0).map(() => newId("key"));
    for (let i = 0; i < keyIds.length; i++) {
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.resources.database.insert(schema.keys).values({
        id: keyIds[i],
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
      });
    }

    const res1 = await h.get<LegacyApisListKeysResponse>({
      url: `/v1/apis/${h.resources.userApi.id}/keys?limit=2`,
      headers: {
        Authorization: `Bearer ${h.resources.rootKey}`,
      },
    });
    expect(res1.status).toEqual(200);

    const res2 = await h.get<LegacyApisListKeysResponse>({
      url: `/v1/apis/${h.resources.userApi.id}/keys?limit=2&offset=2`,
      headers: {
        Authorization: `Bearer ${h.resources.rootKey}`,
      },
    });

    expect(res2.status).toEqual(200);
    const found = new Set<string>();
    for (const key of res1.body.keys) {
      found.add(key.id);
    }
    for (const key of res2.body.keys) {
      found.add(key.id);
    }
    expect(found.size).toEqual(4);
  });
});
