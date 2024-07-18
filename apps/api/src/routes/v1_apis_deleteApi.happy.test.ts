import { randomUUID } from "node:crypto";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { expect, test } from "vitest";
import type { V1ApisDeleteApiRequest, V1ApisDeleteApiResponse } from "./v1_apis_deleteApi";

test("deletes the api", async (t) => {
  const h = await IntegrationHarness.init(t);
  const apiId = newId("api");
  await h.db.primary.insert(schema.apis).values({
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body).toEqual({});

  const found = await h.db.primary.query.apis.findFirst({
    where: (table, { eq, and, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAt)),
  });
  expect(found).toBeUndefined();
});
