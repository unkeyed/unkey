import { expect, test } from "vitest";

import { Harness } from "@/pkg/testutil/harness";
import { newId } from "@unkey/id";
import {
  V1ApisDeleteApiRequest,
  V1ApisDeleteApiResponse,
  registerV1ApisDeleteApi,
} from "./v1_apis_deleteApi";

test("api does not exist", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1ApisDeleteApi);

  const apiId = newId("api");

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
