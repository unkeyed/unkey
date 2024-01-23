import { expect, test } from "vitest";

import { Harness } from "@/pkg/testutil/harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { type V1ApisGetApiResponse, registerV1ApisGetApi } from "./v1_apis_getApi";

test("return the api", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1ApisGetApi);

  const root = await h.createRootKey(["api.*.read_api"]);

  const res = await h.get<V1ApisGetApiResponse>({
    url: `/v1/apis.getApi?apiId=${h.resources.userApi.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body).toEqual({
    id: h.resources.userApi.id,
    name: h.resources.userApi.name,
    workspaceId: h.resources.userApi.workspaceId,
  });
});

test("with ip whitelist", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1ApisGetApi);

  const api = {
    id: newId("api"),
    name: "with ip whitelist",
    workspaceId: h.resources.userWorkspace.id,
    ipWhitelist: JSON.stringify(["127.0.0.1"]),
    createdAt: new Date(),
    deletedAt: null,
  };

  await h.resources.database.insert(schema.apis).values(api);

  const root = await h.createRootKey(["api.*.read_api"]);

  const res = await h.get<V1ApisGetApiResponse>({
    url: `/v1/apis.getApi?apiId=${api.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body).toEqual({
    id: api.id,
    name: api.name,
    workspaceId: api.workspaceId,
  });
});
