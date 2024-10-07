import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";
import { expect, test } from "vitest";

import type { V1KeysWhoAmIRequest, V1KeysWhoAmIResponse } from "./v1_keys_whoami";

test("when the key does not exist", async (t) => {
  const h = await IntegrationHarness.init(t);
  const apiId = newId("api");
  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();

  const root = await h.createRootKey([`api.${apiId}.read_key`]);

  const res = await h.post<V1KeysWhoAmIRequest, V1KeysWhoAmIResponse>({
    url: "/v1/keys.whoami",
    headers: {
      Authorization: `Bearer ${root.key}`,
      "Content-Type": "application/json",
    },
    body: {
      key: key,
    },
  });

  expect(res.status).toEqual(404);
  expect(res.body).toMatchObject({
    error: {
      code: "NOT_FOUND",
      docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
      message: "Key not found",
    },
  });
});
