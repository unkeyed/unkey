import { describe, expect, test } from "vitest";

import type { ErrorResponse } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

describe("with identity", () => {
  describe("with ratelimits", () => {
    describe("missing ratelimit", () => {
      test("returns 400 and a useful error message", async (t) => {
        const h = await IntegrationHarness.init(t);

        const identity = {
          id: newId("test"),
          workspaceId: h.resources.userWorkspace.id,
          externalId: newId("test"),
        };
        await h.db.primary.insert(schema.identities).values(identity);
        await h.db.primary.insert(schema.ratelimits).values({
          id: newId("test"),
          workspaceId: h.resources.userWorkspace.id,
          name: "existing-ratelimit",
          identityId: identity.id,
          limit: 100,
          duration: 60_000,
        });

        const key = await h.createKey({ identityId: identity.id });

        const res = await h.post<any, ErrorResponse>({
          url: "/v1/keys.verifyKey",
          headers: {
            "Content-Type": "application/json",
          },
          body: {
            key: key.key,
            ratelimits: [
              {
                name: "does-not-exist",
              },
            ],
          },
        });

        expect(res.status).toEqual(400);
        expect(res.body.error.message).toMatchInlineSnapshot(
          `"ratelimit "does-not-exist" was requested but does not exist for key "${key.keyId}" nor identity { id: ${identity.id}, externalId: ${identity.externalId}}"`,
        );
      });
    });
  });
});

describe("without identity", () => {
  describe("with ratelimits", () => {
    describe("missing ratelimit", () => {
      test("returns 400 and a useful error message", async (t) => {
        const h = await IntegrationHarness.init(t);

        const key = await h.createKey();

        await h.db.primary.insert(schema.ratelimits).values({
          id: newId("test"),
          workspaceId: h.resources.userWorkspace.id,
          name: "existing-ratelimit",
          keyId: key.keyId,
          limit: 100,
          duration: 60_000,
        });

        const res = await h.post<any, ErrorResponse>({
          url: "/v1/keys.verifyKey",
          headers: {
            "Content-Type": "application/json",
          },
          body: {
            key: key.key,
            ratelimits: [
              {
                name: "does-not-exist",
              },
            ],
          },
        });

        expect(res.status).toEqual(400);
        expect(res.body.error.message).toMatchInlineSnapshot(
          `"ratelimit "does-not-exist" was requested but does not exist for key "${key.keyId}" and there is no identity connected"`,
        );
      });
    });
  });
});
