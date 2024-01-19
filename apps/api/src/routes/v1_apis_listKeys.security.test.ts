import { describe, expect, test } from "vitest";

import { randomUUID } from "crypto";
import type { ErrorResponse } from "@/pkg/errors";
import { Harness } from "@/pkg/testutil/harness";
import { newId } from "@unkey/id";
import { registerV1ApisListKeys } from "./v1_apis_listKeys";

test("without a key", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1ApisListKeys);

  const res = await h.get<ErrorResponse>({
    url: `/v1/apis.listKeys?apiId=${newId("api")}`,
  });

  expect(res.status).toEqual(403);
  expect(res.body).toMatchObject({
    error: {
      code: "UNAUTHORIZED",
      docs: "https://unkey.dev/docs/api-reference/errors/code/UNAUTHORIZED",
      message: "key required",
    },
  });
});

test("with wrong key", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1ApisListKeys);

  const res = await h.get<ErrorResponse>({
    url: `/v1/apis.listKeys?apiId=${newId("api")}`,
    headers: {
      Authorization: "Bearer INVALID_KEY",
    },
  });

  expect(res.status).toEqual(403);
  expect(res.body).toMatchObject({
    error: {
      code: "UNAUTHORIZED",
      docs: "https://unkey.dev/docs/api-reference/errors/code/UNAUTHORIZED",
      message: "you're not allowed to do this",
    },
  });
});

describe("without permission", () => {
  test.each([
    { name: "no roles", roles: [] },
    { name: "wrong roles", roles: [randomUUID(), randomUUID()] },
  ])("$name", async ({ roles }) => {
    const h = await Harness.init();
    h.useRoutes(registerV1ApisListKeys);

    const { key: rootKey } = await h.createRootKey(roles);

    const res = await h.get<ErrorResponse>({
      url: `/v1/apis.listKeys?apiId=${h.resources.userApi.id}`,
      headers: {
        Authorization: `Bearer ${rootKey}`,
      },
    });

    expect(res.status).toEqual(403);
    expect(res.body).toMatchObject({
      error: {
        code: "INSUFFICIENT_PERMISSIONS",
        docs: "https://unkey.dev/docs/api-reference/errors/code/INSUFFICIENT_PERMISSIONS",
        message: "you're not allowed to do this",
      },
    });
  });
});
