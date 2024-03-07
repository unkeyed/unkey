import { randomUUID } from "crypto";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import { runSharedRoleTests } from "@/pkg/testutil/test_route_roles";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { afterEach, beforeEach, describe, expect, test } from "vitest";
import { type V1ApisGetApiResponse, registerV1ApisGetApi } from "./v1_apis_getApi";

let h: RouteHarness;
beforeEach(async () => {
  h = await RouteHarness.init();
  h.useRoutes(registerV1ApisGetApi);
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});

runSharedRoleTests({
  registerHandler: registerV1ApisGetApi,
  prepareRequest: async (rh) => {
    const apiId = newId("api");
    await rh.db.insert(schema.apis).values({
      id: apiId,
      name: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "GET",
      url: `/v1/apis.getApi?apiId=${apiId}`,
    };
  },
});

describe("correct roles", () => {
  test.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard", roles: ["api.*.read_api"] },
    { name: "wildcard and more", roles: ["api.*.read_api", randomUUID()] },
    { name: "specific apiId", roles: [(apiId: string) => `api.${apiId}.read_api`] },
    {
      name: "specific apiId and more",
      roles: [(apiId: string) => `api.${apiId}.read_api`, randomUUID()],
    },
  ])("$name", async ({ roles }) => {
    const apiId = newId("api");
    await h.db.insert(schema.apis).values({
      id: apiId,
      name: randomUUID(),
      workspaceId: h.resources.userWorkspace.id,
    });
    const root = await h.createRootKey(
      roles.map((role) => (typeof role === "string" ? role : role(apiId))),
    );

    const res = await h.get<V1ApisGetApiResponse>({
      url: `/v1/apis.getApi?apiId=${apiId}`,
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });
    expect(res.status).toEqual(200);
  });
});
