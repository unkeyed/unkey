import { describe, expect, test } from "vitest";

import { ErrorResponse } from "@/pkg/errors";
import { Harness } from "@/pkg/testutil/harness";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import {
  type V1KeysGetVerificationsResponse,
  registerV1KeysGetVerifications,
} from "./v1_keys_getVerifications";

describe("when a key does not exist", () => {
  test("returns a 404", async () => {
    const h = await Harness.init();
    h.useRoutes(registerV1KeysGetVerifications);

    const res = await h.get<ErrorResponse>({
      url: "/v1/keys.getVerifications?keyId=INVALID",
      headers: {
        Authorization: `Bearer ${h.resources.rootKey}`,
      },
    });

    expect(res.status).toEqual(404);
    expect(res.body).toEqual({
      error: {
        code: "NOT_FOUND",
        docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
        message: "key INVALID not found",
        // @ts-ignore
        requestId: undefined,
      },
    });
  });
});

describe("when the key exists", () => {
  test("returns an empty verifications array", async () => {
    const h = await Harness.init();
    h.useRoutes(registerV1KeysGetVerifications);

    const keyId = newId("key");
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.resources.database.insert(schema.keys).values({
      id: keyId,
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });

    const res = await h.get<V1KeysGetVerificationsResponse>({
      url: `/v1/keys.getVerifications?keyId=${keyId}`,
      headers: {
        Authorization: `Bearer ${h.resources.rootKey}`,
      },
    });

    expect(res.status).toEqual(200);
    expect(res.body).toEqual({
      verifications: [],
    });
  });

  test("ownerId works too", async () => {
    const h = await Harness.init();
    h.useRoutes(registerV1KeysGetVerifications);

    const ownerId = crypto.randomUUID();
    const keyIds = [newId("key"), newId("key"), newId("key")];
    for (const keyId of keyIds) {
      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.resources.database.insert(schema.keys).values({
        id: keyId,
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        ownerId,
      });
    }

    const res = await h.get<V1KeysGetVerificationsResponse>({
      url: `/v1/keys.getVerifications?ownerId=${ownerId}`,
      headers: {
        Authorization: `Bearer ${h.resources.rootKey}`,
      },
    });

    expect(res.status).toEqual(200);
    expect(res.body).toEqual({
      verifications: [],
    });
  });
});

describe("without a keyId or ownerId", () => {
  test("returns 400", async () => {
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
});
