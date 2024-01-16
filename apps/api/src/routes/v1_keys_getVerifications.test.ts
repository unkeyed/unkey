import { newApp } from "@/pkg/hono/app";
import { describe, expect, test } from "vitest";

import { ErrorResponse } from "@/pkg/errors";
import { init } from "@/pkg/global";
import { unitTestEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { seed } from "@/pkg/testutil/seed";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import {
  type V1KeysGetVerificationsResponse,
  registerV1KeysGetVerifications,
} from "./v1_keys_getVerifications";

describe("when a key does not exist", () => {
  test("returns a 404", async () => {
    const env = unitTestEnv.parse(process.env);
    // @ts-ignore
    init({ env });

    const r = await seed(env);
    const app = newApp();
    registerV1KeysGetVerifications(app);

    const res = await fetchRoute<never, ErrorResponse>(app, {
      method: "GET",
      url: "/v1/keys.getVerifications?keyId=INVALID",
      headers: {
        Authorization: `Bearer ${r.rootKey}`,
      },
    });

    expect(res.status).toEqual(404);
    expect(res.body).toEqual({
      error: {
        code: "NOT_FOUND",
        docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
        message: "key INVALID not found",
        // @ts-ignore
        requestId: undefined,
      },
    });
  });
});

describe("when the key exists", () => {
  test("returns an empty verifications array", async () => {
    const env = unitTestEnv.parse(process.env);
    // @ts-ignore
    init({ env });

    const r = await seed(env);
    const app = newApp();
    registerV1KeysGetVerifications(app);

    const keyId = newId("key");
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await r.database.insert(schema.keys).values({
      id: keyId,
      keyAuthId: r.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: r.userWorkspace.id,
      createdAt: new Date(),
    });

    const res = await fetchRoute<never, V1KeysGetVerificationsResponse>(app, {
      method: "GET",
      url: `/v1/keys.getVerifications?keyId=${keyId}`,
      headers: {
        Authorization: `Bearer ${r.rootKey}`,
      },
    });

    expect(res.status).toEqual(200);
    expect(res.body).toEqual({
      verifications: [],
    });
  });

  test("ownerId works too", async () => {
    const env = unitTestEnv.parse(process.env);
    // @ts-ignore
    init({ env });

    const r = await seed(env);
    const app = newApp();
    registerV1KeysGetVerifications(app);

    const ownerId = crypto.randomUUID();
    const keyIds = [newId("key"), newId("key"), newId("key")];
    for (const keyId of keyIds) {
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await r.database.insert(schema.keys).values({
        id: keyId,
        keyAuthId: r.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: r.userWorkspace.id,
        createdAt: new Date(),
        ownerId,
      });
    }

    const res = await fetchRoute<never, V1KeysGetVerificationsResponse>(app, {
      method: "GET",
      url: `/v1/keys.getVerifications?ownerId=${ownerId}`,
      headers: {
        Authorization: `Bearer ${r.rootKey}`,
      },
    });

    expect(res.status).toEqual(200);
    expect(res.body).toEqual({
      verifications: [],
    });
  });
});

describe("without a keyId or ownerId", () => {
  test("returns 400", async () => {
    const env = unitTestEnv.parse(process.env);
    // @ts-ignore
    init({ env });

    const r = await seed(env);
    const app = newApp();
    registerV1KeysGetVerifications(app);

    const res = await fetchRoute<never, ErrorResponse>(app, {
      method: "GET",
      url: "/v1/keys.getVerifications",
      headers: {
        Authorization: `Bearer ${r.rootKey}`,
      },
    });

    expect(res.status).toEqual(400);
    expect(res.body).toEqual({
      error: {
        code: "BAD_REQUEST",
        docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
        message: "keyId or ownerId must be provided",
        // @ts-ignore
        requestId: undefined,
      },
    });
  });
});
