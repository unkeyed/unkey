import { expect, test } from "bun:test";

import { init } from "@/pkg/global";
import { newApp } from "@/pkg/hono/app";
import { unitTestEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { seed } from "@/pkg/testutil/seed";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import {
  V1KeysUpdateRemainingRequest,
  V1KeysUpdateRemainingResponse,
  registerV1KeysUpdateRemaining,
} from "./v1_keys_updateRemaining";

test("increment", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdateRemaining(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    remaining: 100,
    createdAt: new Date(),
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      keyId: key.id,
      op: "increment",
      value: 10,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body.remaining).toEqual(110);
});

test("decrement", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdateRemaining(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    remaining: 100,
    createdAt: new Date(),
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      keyId: key.id,
      op: "decrement",
      value: 10,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body.remaining).toEqual(90);
});

test("set", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdateRemaining(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    remaining: 100,
    createdAt: new Date(),
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      keyId: key.id,
      op: "set",
      value: 10,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body.remaining).toEqual(10);
});

test("invalid operation", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdateRemaining(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    remaining: 100,
    createdAt: new Date(),
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      keyId: key.id,
      // @ts-ignore This is an invalid operation
      op: "XXX",
      value: 10,
    },
  });

  expect(res.status).toEqual(400);
});
