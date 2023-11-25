import { newApp } from "@/pkg/hono/app";
import { expect, test } from "bun:test";

import { ErrorResponse } from "@/pkg/errors";
import { init } from "@/pkg/global";
import { unitTestEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { seed } from "@/pkg/testutil/seed";
import { schema } from "@unkey/db";
import { eq } from "drizzle-orm";
import {
  V1ApisCreateApiRequest,
  V1ApisCreateApiResponse,
  registerV1ApisCreateApi,
} from "./v1_apis_createApi";

test("creates the api", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });

  const r = await seed(env);
  const app = newApp();
  registerV1ApisCreateApi(app);

  const res = await fetchRoute<V1ApisCreateApiRequest, V1ApisCreateApiResponse>(app, {
    method: "POST",
    url: "/v1/apis.createApi",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      name: "my api",
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body.apiId).toBeDefined();

  const found = await r.database.query.apis.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.apiId),
  });

  expect(found?.name).toBe("my api");
  await r.database.delete(schema.apis).where(eq(schema.apis.id, res.body.apiId));
});

test("creates rejects invalid root key", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });

  await seed(env);
  const app = newApp();
  registerV1ApisCreateApi(app);

  const res = await fetchRoute<V1ApisCreateApiRequest, ErrorResponse>(app, {
    method: "POST",
    url: "/v1/apis.createApi",
    headers: {
      "Content-Type": "application/json",
      Authorization: "Bearer invalidRootKey",
    },
    body: {
      name: "my api",
    },
  });

  expect(res.status).toEqual(403);
  expect(res.body.error.code).toBe("UNAUTHORIZED");
});
