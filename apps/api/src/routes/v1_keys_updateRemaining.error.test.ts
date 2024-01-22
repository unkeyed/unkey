import { expect, test } from "vitest";

import { Harness } from "@/pkg/testutil/harness";
import { newId } from "@unkey/id";
import {
  type V1KeysUpdateRemainingRequest,
  type V1KeysUpdateRemainingResponse,
  registerV1KeysUpdateRemaining,
} from "./v1_keys_updateRemaining";

test("when the key does not exist", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1KeysUpdateRemaining);

  const keyId = newId("key");

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>({
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      op: "set",
      value: 10,
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
