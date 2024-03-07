import { randomUUID } from "crypto";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import { runSharedRoleTests } from "@/pkg/testutil/test_route_roles";
import { afterEach, beforeEach, describe, expect, test } from "vitest";
import {
  V1ApisCreateApiRequest,
  V1ApisCreateApiResponse,
  registerV1ApisCreateApi,
} from "./v1_apis_createApi";

let h: RouteHarness;
beforeEach(async () => {
  h = await RouteHarness.init();
  h.useRoutes(registerV1ApisCreateApi);
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
runSharedRoleTests<V1ApisCreateApiRequest>({
  registerHandler: registerV1ApisCreateApi,
  prepareRequest: () => ({
    method: "POST",
    url: "/v1/apis.createApi",
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      name: randomUUID(),
    },
  }),
});
describe("correct roles", () => {
  test.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard", roles: ["api.*.create_api"] },
    { name: "wildcard and more", roles: ["api.*.create_api", randomUUID()] },
  ])("$name", async ({ roles }) => {
    const root = await h.createRootKey(roles);

    const res = await h.post<V1ApisCreateApiRequest, V1ApisCreateApiResponse>({
      url: "/v1/apis.createApi",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        name: randomUUID(),
      },
    });
    expect(res.status).toEqual(200);

    const found = await h.db.query.apis.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.apiId),
    });
    expect(found).toBeDefined();
  });
});
