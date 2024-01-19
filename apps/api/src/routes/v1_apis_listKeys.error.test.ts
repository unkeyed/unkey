import { describe, expect, test } from "vitest";

import type { ErrorResponse } from "@/pkg/errors";
import { Harness } from "@/pkg/testutil/harness";
import { newId } from "@unkey/id";
import { registerV1ApisListKeys } from "./v1_apis_listKeys";

describe("when the api does not exist", () => {
  test("returns 404", async () => {
    const h = await Harness.init();
    h.useRoutes(registerV1ApisListKeys);

    const apiId = newId("api");

    const { key: rootKey } = await h.createRootKey([`api.${apiId}.read_api`]);

    const res = await h.get<ErrorResponse>({
      url: `/v1/apis.listKeys?apiId=${apiId}`,
      headers: {
        Authorization: `Bearer ${rootKey}`,
      },
    });

    expect(res.status).toEqual(404);
    expect(res.body).toMatchObject({
      error: {
        code: "NOT_FOUND",
        docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
        message: `api ${apiId} not found`,
      },
    });
  });
});
