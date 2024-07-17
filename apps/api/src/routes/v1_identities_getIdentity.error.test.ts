import { expect, test } from "vitest";

import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { V1IdentitiesGetIdentityResponse } from "./v1_identities_getIdentity";

test("identity does not exist", async (t) => {
  const h = await IntegrationHarness.init(t);
  const identityId = newId("test");

  const root = await h.createRootKey(["identity.*.read_identity"]);

  const res = await h.get<V1IdentitiesGetIdentityResponse>({
    url: `/v1/identities.getIdentity?identityId=${identityId}`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toEqual(404);
  expect(res.body).toMatchObject({
    error: {
      code: "NOT_FOUND",
      docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
      message: `identity ${identityId} not found`,
    },
  });
});
