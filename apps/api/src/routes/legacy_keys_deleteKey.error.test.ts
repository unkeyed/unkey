import { expect, test } from "vitest";

import type { ErrorResponse } from "@/pkg/errors";
import { Harness } from "@/pkg/testutil/harness";
import { newId } from "@unkey/id";
import { registerLegacyKeysDelete } from "./legacy_keys_deleteKey";
test("key not found", async () => {
  const h = await Harness.init();
  h.useRoutes(registerLegacyKeysDelete);

  const keyId = newId("key");

  const { key: rootKey } = await h.createRootKey(["*"]);

  const res = await h.delete<ErrorResponse>({
    url: `/v1/keys/${keyId}`,
    headers: {
      Authorization: `Bearer ${rootKey}`,
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
