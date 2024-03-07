import { afterEach, beforeEach, expect, test } from "vitest";

import type { ErrorResponse } from "@/pkg/errors";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import { newId } from "@unkey/id";
import { registerV1ApisListKeys } from "./v1_apis_listKeys";

let h: RouteHarness;
beforeEach(async () => {
  h = await RouteHarness.init();
  h.useRoutes(registerV1ApisListKeys);
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
test("api not found", async () => {
  const apiId = newId("api");

  const { key: rootKey } = await h.createRootKey([
    `api.${apiId}.read_api`,
    `api.${apiId}.read_key`,
  ]);

  const res = await h.get<ErrorResponse>({
    url: `/v1/apis.listKeys?apiId=${apiId}`,
    headers: {
      Authorization: `Bearer ${rootKey}`,
    },
  });

  expect(res.status).toEqual(404);
  expect(res.body).toMatchObject({
    error: {
      code: "NOT_FOUND",
      docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
      message: `api ${apiId} not found`,
    },
  });
});
