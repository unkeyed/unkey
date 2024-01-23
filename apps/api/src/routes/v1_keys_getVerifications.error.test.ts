import { expect, test } from "vitest";

import type { ErrorResponse } from "@/pkg/errors";
import { Harness } from "@/pkg/testutil/harness";
import { newId } from "@unkey/id";
import { registerV1KeysGetVerifications } from "./v1_keys_getVerifications";

test("when the key does not exist", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1KeysGetVerifications);

  const keyId = newId("api");

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.read_key`]);

  const res = await h.get<ErrorResponse>({
    url: `/v1/keys.getVerifications?keyId=${keyId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toEqual(404);
  expect(res.body).toMatchObject({
    error: {
      code: "NOT_FOUND",
      docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
      message: `key ${keyId} not found`,
    },
  });
});

test("without keyId or ownerId", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1KeysGetVerifications);

  const res = await h.get<ErrorResponse>({
    url: "/v1/keys.getVerifications",
    headers: {
      Authorization: `Bearer ${h.resources.rootKey}`,
    },
  });

  expect(res.status).toEqual(400);
  expect(res.body).toEqual({
    error: {
      code: "BAD_REQUEST",
      docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
      message: "keyId or ownerId must be provided",
      // @ts-ignore
      requestId: undefined,
    },
  });
});
