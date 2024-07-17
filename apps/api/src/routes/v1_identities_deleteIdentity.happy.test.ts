import { randomUUID } from "node:crypto";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { expect, test } from "vitest";
import type {
  V1IdentitiesDeleteIdentityRequest,
  V1IdentitiesDeleteIdentityResponse,
} from "./v1_identities_deleteIdentity";

test("deletes the identity", async (t) => {
  const h = await IntegrationHarness.init(t);
  const identityId = newId("test");
  await h.db.primary.insert(schema.identities).values({
    id: identityId,
    externalId: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  });

  const root = await h.createRootKey([`identity.${identityId}.delete_identity`]);
  const res = await h.post<V1IdentitiesDeleteIdentityRequest, V1IdentitiesDeleteIdentityResponse>({
    url: "/v1/identities.deleteIdentity",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      identityId,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body).toEqual({});

  const found = await h.db.primary.query.identities.findFirst({
    where: (table, { eq }) => eq(table.id, identityId),
  });
  expect(found).toBeUndefined();
});
