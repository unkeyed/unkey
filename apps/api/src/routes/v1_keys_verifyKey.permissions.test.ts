import { describe, expect, test } from "vitest";

import { ErrorResponse } from "@/pkg/errors";
import { Harness } from "@/pkg/testutil/harness";
import { and, schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { NestedQuery, buildQuery } from "@unkey/rbac";
import {
  V1KeysVerifyKeyRequest,
  V1KeysVerifyKeyResponse,
  registerV1KeysVerifyKey,
} from "./v1_keys_verifyKey";

type TestCase = {
  name: string;
  roles: {
    name: string;
    permissions: string[];
  }[];
  query: NestedQuery | undefined;
  expected: {
    status: number;
    valid: boolean;
  };
};

test.each<TestCase>([
  {
    name: "No Roles and no query",
    roles: [],
    query: undefined,
    expected: { status: 200, valid: true },
  },
  {
    name: "No query",
    roles: [
      {
        name: newId("test"),
        permissions: [],
      },
    ],
    query: undefined,
    expected: { status: 200, valid: true },
  },
  {
    name: "Single role, single permission",
    roles: [
      {
        name: newId("test"),
        permissions: ["p1"],
      },
    ],
    query: buildQuery(({ or }) => or("p1")).query,
    expected: { status: 200, valid: true },
  },
])("$name", async ({ name, roles, query, expected }) => {
  const h = await Harness.init();
  h.useRoutes(registerV1KeysVerifyKey);

  const { key } = await h.createKey({ roles });

  const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: "/v1/keys.verifyKey",
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      key,
      apiId: h.resources.userApi.id,
      authorization: query
        ? {
            permissions: query,
          }
        : undefined,
    },
  });
  expect(res.status).toEqual(expected.status);
  expect(res.body.valid, `key is not valid, received body: ${JSON.stringify(res.body)}`).toBe(
    expected.valid,
  );
});
