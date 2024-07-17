import { expect, test } from "vitest";

import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type {
  V1IdentitiesDeleteIdentityRequest,
  V1IdentitiesDeleteIdentityResponse,
} from "./v1_identities_deleteIdentity";

test("identity does not exist", async (t) => {
  const h = await IntegrationHarness.init(t);
  const identityId = newId("test");

  const { key: rootKey } = await h.createRootKey(["identity.*.delete_identity"]);

  const res = await h.post<V1IdentitiesDeleteIdentityRequest, V1IdentitiesDeleteIdentityResponse>({
    url: "/v1/identities.deleteIdentity",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      identityId,
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
