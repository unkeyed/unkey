import { expect, test } from "vitest";

import type { ErrorResponse } from "@/pkg/errors";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

test("when the key does not exist", async (t) => {
  const h = await IntegrationHarness.init(t);
  const keyId = newId("api");

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.read_key`]);

  const res = await h.get<ErrorResponse>({
    url: `/v1/keys.getVerifications?keyId=${keyId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toEqual(404);
  expect(res.body.error.code).toEqual("NOT_FOUND");
  expect(res.body.error.docs).toEqual("https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND");
  expect(res.body.error.message).toEqual(`key ${keyId} not found`);
  expect(res.body.error.requestId).toMatch(/^req_[a-zA-Z0-9]+$/);
});

test("without keyId or ownerId", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key } = await h.createRootKey(["*"]);
  const res = await h.get<ErrorResponse>({
    url: "/v1/keys.getVerifications",
    headers: {
      Authorization: `Bearer ${key}`,
    },
  });

  expect(res.status).toEqual(400);
  expect(res.body.error.code).toEqual("BAD_REQUEST");
  expect(res.body.error.docs).toEqual(
    "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
  );
  expect(res.body.error.message).toEqual("keyId or ownerId must be provided");
  expect(res.body.error.requestId).toMatch(/^req_[a-zA-Z0-9]+$/);
});
