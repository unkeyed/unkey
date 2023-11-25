import { describe, expect, test } from "bun:test";

import { init } from "@/pkg/global";
import { newApp } from "@/pkg/hono/app";
import { unitTestEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { seed } from "@/pkg/testutil/seed";
import { sha256 } from "@unkey/hash";

import { ErrorResponse } from "@/pkg/errors";
import {
  V1KeysCreateKeyRequest,
  V1KeysCreateKeyResponse,
  registerV1KeysCreateKey,
} from "./v1_keys_createKey";

describe("simple", () => {
  test("creates key", async () => {
    const env = unitTestEnv.parse(process.env);
    // @ts-ignore
    init({ env });
    const app = newApp();
    registerV1KeysCreateKey(app);

    const r = await seed(env);

    const res = await fetchRoute<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>(app, {
      method: "POST",
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${r.rootKey}`,
      },
      body: {
        byteLength: 16,
        apiId: r.userApi.id,
      },
    });

    expect(res.status).toEqual(200);

    const found = await r.database.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
    });
    expect(found).toBeDefined();
    expect(found!.hash).toEqual(await sha256(res.body.key));
  });
});

describe("wrong ratelimit type", () => {
  test("reject the request", async () => {
    const env = unitTestEnv.parse(process.env);
    // @ts-ignore
    init({ env });
    const app = newApp();
    registerV1KeysCreateKey(app);

    const r = await seed(env);

    const res = await fetchRoute<V1KeysCreateKeyRequest, ErrorResponse>(app, {
      method: "POST",
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${r.rootKey}`,
      },
      body: {
        byteLength: 16,
        apiId: r.userApi.id,
        ratelimit: {
          // @ts-expect-error
          type: "x",
        },
      },
    });

    expect(res.status).toEqual(400);
    expect(res.body.error.code).toEqual("BAD_REQUEST");
  });
});

describe("with prefix", () => {
  test("start includes prefix", async () => {
    const env = unitTestEnv.parse(process.env);
    // @ts-ignore
    init({ env });
    const app = newApp();
    registerV1KeysCreateKey(app);

    const r = await seed(env);

    const res = await fetchRoute<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>(app, {
      method: "POST",
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${r.rootKey}`,
      },
      body: {
        byteLength: 16,
        apiId: r.userApi.id,
        prefix: "prefix",
      },
    });

    expect(res.status).toEqual(200);

    const key = await r.database.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.keyId),
    });
    expect(key).toBeDefined();
    expect(key!.start).toStartWith("prefix_");
  });
});
