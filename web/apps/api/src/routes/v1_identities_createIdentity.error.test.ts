import type { ErrorResponse } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";
import { describe, expect, test } from "vitest";
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
describe("when identity exists already", () => {
  test("should return correct code and message", async (t) => {
    const h = await IntegrationHarness.init(t);
    const { key: rootKey } = await h.createRootKey(["*"]);

    const externalId = newId("test");
    await h.db.primary.insert(schema.identities).values({
      id: newId("test"),
      workspaceId: h.resources.userWorkspace.id,
      externalId,
    });

    const res = await h.post<V1IdentitiesCreateIdentityRequest, ErrorResponse>({
      url: "/v1/identities.createIdentity",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${rootKey}`,
      },
      body: {
        externalId: externalId,
      },
    });

    expect(res.status, `expected 409, received: ${JSON.stringify(res, null, 2)}`).toEqual(409);
    expect(res.body).toMatchObject({
      error: {
        code: "CONFLICT",
        docs: "https://unkey.dev/docs/api-reference/errors/code/CONFLICT",
        message: `Identity with externalId "${externalId}" already exists in this workspace`,
      },
    });
  });
});
