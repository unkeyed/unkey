import { describe, expect, test } from "vitest";

import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type {
  V1IdentitiesCreateIdentityRequest,
  V1IdentitiesCreateIdentityResponse,
} from "./v1_identities_createIdentity";

describe.each([
  { name: "empty externalId", externalId: "" },
  { name: "short externalId", externalId: "ab" },
])("$name", ({ externalId }) => {
  test("reject", async (t) => {
    const h = await IntegrationHarness.init(t);
    const { key: rootKey } = await h.createRootKey(["*"]);

    const res = await h.post<V1IdentitiesCreateIdentityRequest, V1IdentitiesCreateIdentityResponse>(
      {
        url: "/v1/identities.createIdentity",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${rootKey}`,
        },
        body: {
          externalId: externalId,
        },
      },
    );

    expect(res.status).toEqual(400);
    expect(res.body).toMatchObject({
      error: {
        code: "BAD_REQUEST",
        docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
        message: "externalId: String must contain at least 3 character(s)",
      },
    });
  });
});
