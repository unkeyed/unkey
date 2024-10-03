import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type {
  V1RatelimitSetOverrideRequest,
  V1RatelimitSetOverrideResponse
} from "./v1_ratelimit_setOverride";

test("Set ratelimit override", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["ratelimit.*.set_override"]);
  const overrideId = newId("test");
  const identifier = randomUUID();
  const namespaceId = newId("test");


  const override = {
    overrideId: overrideId,
    namespaceId: newId("test"),
    identifier: identifier,
    limit: 1,
    duration: 60000,
    async: false,
  }
  const namespace = {
    id: namespaceId,
    workspaceId: h.resources.userWorkspace.id,
    name: "namespace",
    createdAt: new Date(),
  };
  
  await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);
  
  const res = await h.post<V1RatelimitSetOverrideRequest, V1RatelimitSetOverrideResponse>({
    url: "/v1/ratelimit.setOverride",
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
    body: { 
      namespaceId: "rlns_2gXQ4vGpBYxzQfNDFB9NyqBkoQhX",
      identifier: "asdfadsfs",
      limit: 10,
      duration: 65000,
      async: false
    }
  });


  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);


  // expect(resInit.status, `expected 200, received: ${JSON.stringify(resInit, null, 2)}`).toBe(200);
  // expect(resInit.id).toBe(overrideId);
  // expect(resInit.namespaceId).toEqual(namespaceId);
  // expect(resInit?.identifier).toEqual(identifier);
  // expect(resInit?.limit).toEqual(override.limit);
  // expect(resInit?.duration).toEqual(override.duration);
  // expect(resInit?.async).toEqual(override.async);

  // const resUpdate = await h.post<V1RatelimitSetOverrideRequest, V1RatelimitSetOverrideResponse>({
  //   url: `/v1/ratelimits.setOverride`,
  //   headers: {
  //     Authorization: `Bearer ${root.key}`,
  //   },
  //   body: {
  //     namespaceId: namespaceId,
  //     identifier: identifier,
  //     limit: override.limit + 1,
  //     duration: override.duration - 5000,
  //     async: !override.async,
  //   }

  // });
  // expect(resUpdate.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  // const resNew = await h.db.primary.query.ratelimitOverrides.findFirst({
  //   where: (table, { eq }) => eq(table.id, overrideId),
  // });

  // expect(resNew?.id).toBe(overrideId);
  // expect(resNew?.namespaceId).toEqual(namespaceId);
  // expect(resNew?.identifier).toEqual(identifier);
  // expect(resNew?.limit).toEqual(override.limit + 1);
  // expect(resNew?.duration).toEqual(override.duration - 5000);
  // expect(resNew?.async).toEqual(!override.async);
});
