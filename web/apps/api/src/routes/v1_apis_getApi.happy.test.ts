import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { V1ApisGetApiResponse } from "./v1_apis_getApi";

test("return the api", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["api.*.read_api"]);

  const res = await h.get<V1ApisGetApiResponse>({
    url: `/v1/apis.getApi?apiId=${h.resources.userApi.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body).toEqual({
    id: h.resources.userApi.id,
    name: h.resources.userApi.name,
    workspaceId: h.resources.userApi.workspaceId,
  });
});

test("with ip whitelist", async (t) => {
  const h = await IntegrationHarness.init(t);
  const api = {
    id: newId("api"),
    name: "with ip whitelist",
    workspaceId: h.resources.userWorkspace.id,
    ipWhitelist: ["127.0.0.1"].join(","),
    createdAtM: Date.now(),
    deletedAtM: null,
  };

  await h.db.primary.insert(schema.apis).values(api);

  const root = await h.createRootKey(["api.*.read_api"]);

  const res = await h.get<V1ApisGetApiResponse>({
    url: `/v1/apis.getApi?apiId=${api.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body).toEqual({
    id: api.id,
    name: api.name,
    workspaceId: api.workspaceId,
  });
});
