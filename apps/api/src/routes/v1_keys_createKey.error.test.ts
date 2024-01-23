import { expect, test } from "vitest";

import { randomUUID } from "crypto";
import type { ErrorResponse } from "@/pkg/errors";
import { Harness } from "@/pkg/testutil/harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import {
  V1KeysCreateKeyRequest,
  V1KeysCreateKeyResponse,
  registerV1KeysCreateKey,
} from "./v1_keys_createKey";

test("when the api does not exist", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1KeysCreateKey);

  const apiId = newId("api");

  const root = await h.createRootKey([`api.${apiId}.create_key`]);
  /* The code snippet is making a POST request to the "/v1/keys.createKey" endpoint with the specified headers. It is using the `h.post` method from the `Harness` instance to send the request. The generic types `<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>` specify the request payload and response types respectively. */

  const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      byteLength: 16,
      apiId,
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

test("when the api has no keyAuth", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1KeysCreateKey);

  const apiId = newId("api");
  await h.db.insert(schema.apis).values({
    id: apiId,
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  });

  const root = await h.createRootKey([`api.${apiId}.create_key`]);

  const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      byteLength: 16,
      apiId,
    },
  });
  expect(res.status).toEqual(412);
  expect(res.body).toMatchObject({
    error: {
      code: "PRECONDITION_FAILED",
      docs: "https://unkey.dev/docs/api-reference/errors/code/PRECONDITION_FAILED",
      message: `api ${apiId} is not setup to handle keys`,
    },
  });
});

test("reject invalid ratelimit config", async () => {
  const h = await Harness.init();
  h.useRoutes(registerV1KeysCreateKey);

  const res = await h.post<V1KeysCreateKeyRequest, ErrorResponse>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${h.resources.rootKey}`,
    },
    body: {
      byteLength: 16,
      apiId: h.resources.userApi.id,
      ratelimit: {
        // @ts-expect-error
        type: "x",
      },
    },
  });

  expect(res.status).toEqual(400);
  expect(res.body.error.code).toEqual("BAD_REQUEST");
});
