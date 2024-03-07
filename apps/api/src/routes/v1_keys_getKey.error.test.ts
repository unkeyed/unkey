import { afterEach, beforeEach, expect, test } from "vitest";

import type { ErrorResponse } from "@/pkg/errors";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import { newId } from "@unkey/id";
import { registerV1KeysGetKey } from "./v1_keys_getKey";

let h: RouteHarness;
beforeEach(async () => {
  h = await RouteHarness.init();
  h.useRoutes(registerV1KeysGetKey);
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
test("when the key does not exist", async () => {
  const apiId = newId("api");
  const keyId = newId("api");

  const root = await h.createRootKey([`api.${apiId}.read_key`]);

  const res = await h.get<ErrorResponse>({
    url: `/v1/keys.getKey?keyId=${keyId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
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
