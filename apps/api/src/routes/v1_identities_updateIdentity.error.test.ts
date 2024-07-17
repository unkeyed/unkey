import { expect, test } from "vitest";

import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type {
  V1IdentitiesUpdateIdentityRequest,
  V1IdentitiesUpdateIdentityResponse,
} from "./v1_identities_updateIdentity";

test("empty identityId", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key: rootKey } = await h.createRootKey(["identity.*.update_identity"]);

  const identity = {
    id: newId("test"),
    workspaceId: h.resources.userWorkspace.id,
    externalId: randomUUID(),
  };

  await h.db.primary.insert(schema.identities).values(identity);

  const res = await h.post<V1IdentitiesUpdateIdentityRequest, V1IdentitiesUpdateIdentityResponse>({
    url: "/v1/identities.updateIdentity",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {},
  });

  expect(res.status).toEqual(400);
  expect(res.body).toMatchObject({
    error: {
      code: "BAD_REQUEST",
      docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
      message: "Provide either identityId or externalId",
    },
  });
});
