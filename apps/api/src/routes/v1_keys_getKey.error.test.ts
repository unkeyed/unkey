import { expect, test } from "vitest";

import type { ErrorResponse } from "@/pkg/errors";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import { newId } from "@unkey/id";

test("when the key does not exist", async () => {
  const h = await RouteHarness.init();
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
