import { expect, test } from "vitest";

import { Harness } from "@/pkg/testutil/harness";
import {
  V1ApisCreateApiRequest,
  V1ApisCreateApiResponse,
  registerV1ApisCreateApi,
} from "./v1_apis_createApi";

test.each([
  { name: "empty name", apiName: "" },
  { name: "short name", apiName: "ab" },
])("$name", async ({ apiName }) => {
  const h = await Harness.init();
  h.useRoutes(registerV1ApisCreateApi);

  const { key: rootKey } = await h.createRootKey(["*"]);

  const res = await h.post<V1ApisCreateApiRequest, V1ApisCreateApiResponse>({
    url: "/v1/apis.createApi",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      name: apiName,
    },
  });

  expect(res.status).toEqual(400);
  expect(res.body).toMatchObject({
    error: {
      code: "BAD_REQUEST",
      docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
      message:
        'too_small: name: String must contain at least 3 character(s), See "https://unkey.dev/docs/api-reference" for more details',
    },
  });
});
