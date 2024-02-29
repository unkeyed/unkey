import { randomUUID } from "crypto";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { afterEach, beforeEach, expect, test } from "vitest";
import {
  V1ApisDeleteApiRequest,
  V1ApisDeleteApiResponse,
  registerV1ApisDeleteApi,
} from "./v1_apis_deleteApi";

let h: RouteHarness;
beforeEach(async () => {
  h = new RouteHarness();
  h.useRoutes(registerV1ApisDeleteApi);
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
test("deletes the api", async () => {
  const apiId = newId("api");
  await h.db.insert(schema.apis).values({
    id: apiId,
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  });

  const root = await h.createRootKey([`api.${apiId}.delete_api`]);
  const res = await h.post<V1ApisDeleteApiRequest, V1ApisDeleteApiResponse>({
    url: "/v1/apis.deleteApi",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body).toEqual({});

  const found = await h.db.query.apis.findFirst({
    where: (table, { eq, and, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAt)),
  });
  expect(found).toBeUndefined();
});
