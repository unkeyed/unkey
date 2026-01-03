import { expect, test } from "vitest";

import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { V1ApisDeleteApiRequest, V1ApisDeleteApiResponse } from "./v1_apis_deleteApi";

test("api does not exist", async (t) => {
  const h = await IntegrationHarness.init(t);
  const apiId = newId("test");

  const { key: rootKey } = await h.createRootKey(["*"]);

  const res = await h.post<V1ApisDeleteApiRequest, V1ApisDeleteApiResponse>({
    url: "/v1/apis.deleteApi",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      apiId,
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
