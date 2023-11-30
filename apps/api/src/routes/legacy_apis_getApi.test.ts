import { newApp } from "@/pkg/hono/app";
import { describe, expect, test } from "bun:test";

import { ErrorResponse } from "@/pkg/errors";
import { init } from "@/pkg/global";
import { unitTestEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { seed } from "@/pkg/testutil/seed";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { type LegacyApisGetApiResponse, registerLegacyApisGetApi } from "./legacy_apis_getApi";

describe("when api exists", () => {
  describe("basic", () => {
    test("returns the api", async () => {
      const env = unitTestEnv.parse(process.env);
      // @ts-ignore
      init({ env });

      const r = await seed(env);
      const app = newApp();
      registerLegacyApisGetApi(app);

      const res = await fetchRoute<never, LegacyApisGetApiResponse>(app, {
        method: "GET",
        url: `/v1/apis/${r.userApi.id}`,
        headers: {
          Authorization: `Bearer ${r.rootKey}`,
        },
      });

      expect(res.status).toEqual(200);
      expect(res.body).toEqual({
        id: r.userApi.id,
        name: r.userApi.name,
        workspaceId: r.userApi.workspaceId,
      });
    });
  });

  describe("with ip whitelist", () => {
    test("returns the ip whitelist", async () => {
      const env = unitTestEnv.parse(process.env);
      // @ts-ignore
      init({ env });

      const r = await seed(env);

      const api = {
        id: newId("api"),
        name: "with ip whitelist",
        workspaceId: r.userWorkspace.id,
        ipWhitelist: JSON.stringify(["127.0.0.1"]),
      };

      await r.database.insert(schema.apis).values(api);

      const app = newApp();
      registerLegacyApisGetApi(app);

      const res = await fetchRoute<never, LegacyApisGetApiResponse>(app, {
        method: "GET",
        url: `/v1/apis/${api.id}`,
        headers: {
          Authorization: `Bearer ${r.rootKey}`,
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
describe("when api does not exist", () => {
  test("returns an error", async () => {
    const env = unitTestEnv.parse(process.env);
    // @ts-ignore
    init({ env });

    const r = await seed(env);
    const app = newApp();
    registerLegacyApisGetApi(app);

    const fakeApiId = newId("api");

    const res = await fetchRoute<never, ErrorResponse>(app, {
      method: "GET",
      url: `/v1/apis/${fakeApiId}`,
      headers: {
        Authorization: `Bearer ${r.rootKey}`,
      },
    });

    expect(res.status).toEqual(404);
    expect(res.body.error.code).toEqual("NOT_FOUND");
  });
});
