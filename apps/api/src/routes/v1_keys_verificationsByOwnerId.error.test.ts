import { expect, test } from "vitest";

import type { ErrorResponse } from "@/pkg/errors";
import { RouteHarness } from "src/pkg/testutil/route-harness";

test("when the ownerId does not exist", async (t) => {
  const h = await RouteHarness.init(t);
  const ownerId = crypto.randomUUID();
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.read_key`]);

  const res = await h.get<ErrorResponse>({
    url: `/v1/analytics.getByOwnerId?ownerId=${ownerId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toEqual(404);
  expect(res.body.error.code).toEqual("NOT_FOUND");
  expect(res.body.error.docs).toEqual("https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND");
  expect(res.body.error.message).toEqual(`ownerId ${ownerId} not found`);
  expect(res.body.error.requestId).toMatch(/^req_[a-zA-Z0-9]+$/);
});

test("without ownerId", async (t) => {
  const h = await RouteHarness.init(t);
  const { key } = await h.createRootKey(["*"]);
  const res = await h.get<ErrorResponse>({
    url: "/v1/analytics.getByOwnerId",
    headers: {
      Authorization: `Bearer ${key}`,
    },
  });

  expect(res.status).toEqual(400);
  expect(res.body.error.code).toEqual("BAD_REQUEST");
  expect(res.body.error.message).toEqual(
    `invalid_type: ownerId: Required, See "https://unkey.dev/docs/api-reference" for more details`,
  );
  expect(res.body.error.docs).toEqual(
    "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
  );

  expect(res.body.error.requestId).toMatch(/^req_[a-zA-Z0-9]+$/);
});
