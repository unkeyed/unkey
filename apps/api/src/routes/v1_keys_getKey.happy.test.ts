import { RouteHarness } from "@/pkg/testutil/route-harness";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { afterEach, beforeEach, expect, test } from "vitest";
import { V1KeysGetKeyResponse, registerV1KeysGetKey } from "./v1_keys_getKey";

let h: RouteHarness;
beforeEach(async () => {
  h = new RouteHarness();
  h.useRoutes(registerV1KeysGetKey);
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
test("returns 200", async () => {
  const root = await h.createRootKey(["api.*.read_key"]);
  const key = {
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
  };
  await h.db.insert(schema.keys).values(key);

  const res = await h.get<V1KeysGetKeyResponse>({
    url: `/v1/keys.getKey?keyId=${key.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });
  expect(res.status).toEqual(200);

  expect(res.body.id).toEqual(key.id);
  expect(res.body.apiId).toEqual(h.resources.userApi.id);
  expect(res.body.workspaceId).toEqual(key.workspaceId);
  expect(res.body.name).toEqual(key.name);
  expect(res.body.start).toEqual(key.start);
  expect(res.body.createdAt).toEqual(key.createdAt.getTime());
});
