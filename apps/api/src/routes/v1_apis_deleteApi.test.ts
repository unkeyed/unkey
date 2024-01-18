import { Harness } from "@/pkg/testutil/harness";
import { expect, test } from "vitest";
import {
  V1ApisDeleteApiRequest,
  V1ApisDeleteApiResponse,
  registerV1ApisDeleteApi,
} from "./v1_apis_deleteApi";

test("deletes the api", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1ApisDeleteApi);

  const res = await h.post<V1ApisDeleteApiRequest, V1ApisDeleteApiResponse>({
    url: "/v1/apis.deleteApi",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${h.resources.rootKey}`,
    },
    body: {
      apiId: h.resources.userApi.id,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body).toEqual({});

  const found = await h.db.query.apis.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, h.resources.userApi.id), isNull(table.deletedAt)),
  });
  expect(found).toBeUndefined();
});
