import { afterEach, beforeEach, describe, expect, test } from "vitest";

import { ErrorResponse } from "@/pkg/errors";
import { fetchRoute } from "@/pkg/testutil/request";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { type LegacyApisGetApiResponse, registerLegacyApisGetApi } from "./legacy_apis_getApi";

let h: RouteHarness;
beforeEach(async () => {
  h = new RouteHarness();
  h.useRoutes(registerLegacyApisGetApi);
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
describe("when api exists", () => {
  describe("with a `*` role", () => {
    test("returns the api", async () => {
      const rootKey = await h.createRootKey(["*"]);

      const res = await fetchRoute<never, LegacyApisGetApiResponse>(h.app, {
        method: "GET",
        url: `/v1/apis/${h.resources.userApi.id}`,
        headers: {
          Authorization: `Bearer ${rootKey.key}`,
        },
      });

      expect(res.status).toEqual(200);
      expect(res.body).toEqual({
        id: h.resources.userApi.id,
        name: h.resources.userApi.name,
        workspaceId: h.resources.userApi.workspaceId,
      });
    });
  });

  describe("with ip whitelist", () => {
    describe("with `*` role", () => {
      test("returns the ip whitelist", async () => {
        const api = {
          id: newId("api"),
          name: "with ip whitelist",
          workspaceId: h.resources.userWorkspace.id,
          ipWhitelist: JSON.stringify(["127.0.0.1"]),
          createdAt: new Date(),
          deletedAt: null,
        };

        await h.db.insert(schema.apis).values(api);

        const rootKey = await h.createRootKey(["*"]);

        const res = await h.get<LegacyApisGetApiResponse>({
          url: `/v1/apis/${api.id}`,
          headers: {
            Authorization: `Bearer ${rootKey.key}`,
          },
        });

        expect(res.status).toEqual(200);
        expect(res.body).toEqual({
          id: api.id,
          name: api.name,
          workspaceId: api.workspaceId,
        });
      });
    });
  });
});
describe("when api does not exist", () => {
  test("returns an error", async () => {
    const rootKey = await h.createRootKey(["*"]);

    const fakeApiId = newId("api");

    const res = await h.get<ErrorResponse>({
      url: `/v1/apis/${fakeApiId}`,
      headers: {
        Authorization: `Bearer ${rootKey.key}`,
      },
    });

    expect(res.status).toEqual(404);
    expect(res.body.error.code).toEqual("NOT_FOUND");
  });
});

describe("without roles", () => {
  describe("when api exists", () => {
    describe("basic", () => {
      test("returns the api", async () => {
        const { key: rootKey } = await h.createRootKey([]);

        const res = await h.get<ErrorResponse>({
          url: `/v1/apis/${h.resources.userApi.id}`,
          headers: {
            Authorization: `Bearer ${rootKey}`,
          },
        });

        expect(res.status).toEqual(403);
        expect(res.body.error.code).toEqual("INSUFFICIENT_PERMISSIONS");
      });
    });
  });
  describe("when api does not exist", () => {
    test("returns an error", async () => {
      const fakeApiId = newId("api");
      const { key: rootKey } = await h.createRootKey(["*"]);

      const res = await h.get<ErrorResponse>({
        url: `/v1/apis/${fakeApiId}`,
        headers: {
          Authorization: `Bearer ${rootKey}`,
        },
      });

      expect(res.status).toEqual(404);
      expect(res.body.error.code).toEqual("NOT_FOUND");
    });
  });
});
