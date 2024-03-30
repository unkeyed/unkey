import { expect, test } from "vitest";

import { newId } from "@unkey/id";
import { RouteHarness } from "src/pkg/testutil/route-harness";
import type { V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse } from "./v1_keys_updateKey";

test("when the key does not exist", async (t) => {
  const h = await RouteHarness.init(t);
  const keyId = newId("key");

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      enabled: false,
    },
  });
  expect(res.status).toEqual(404);
  expect(res.body).toMatchObject({
    error: {
      code: "NOT_FOUND",
      docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
      message: `key ${keyId} not found`,
    },
  });
});
