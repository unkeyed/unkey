import { randomUUID } from "crypto";
import { Harness } from "@/pkg/testutil/harness";
import { runSharedRoleTests } from "@/pkg/testutil/test_route_roles";
import { describe, expect, test } from "vitest";
import {
  V1ApisCreateApiRequest,
  V1ApisCreateApiResponse,
  registerV1ApisCreateApi,
} from "./v1_apis_createApi";
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
    const h = await Harness.init();
    h.useRoutes(registerV1ApisCreateApi);

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

    const found = await h.resources.database.query.apis.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.apiId),
    });
    expect(found).toBeDefined();
  });
});
