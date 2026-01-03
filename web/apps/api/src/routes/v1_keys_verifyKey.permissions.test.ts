import { newId } from "@unkey/id";
import { type PermissionQuery, buildQuery } from "@unkey/rbac";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "./v1_keys_verifyKey";

type TestCase = {
  name: string;
  roles: {
    name: string;
    permissions: string[];
  }[];
  query: PermissionQuery | undefined;
  expected: {
    status: number;
    valid: boolean;
  };
};

describe.each<TestCase>([
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
    query: buildQuery(({ or }) => or("p1")),
    expected: { status: 200, valid: true },
  },
  {
    name: "No roles, but required",
    roles: [],
    query: buildQuery(({ or }) => or("p1")),
    expected: { status: 200, valid: false },
  },
  {
    name: "nested of 'and' and 'or'",
    roles: [
      {
        name: "r1",
        permissions: ["p1", "p2", "p3"],
      },
      {
        name: "r2",
        permissions: ["p1", "p4", "p5"],
      },
    ],
    query: buildQuery(({ or, and }) => and("p1", or("p2", "p6", and("p4", "p2")))),
    expected: { status: 200, valid: true },
  },
  {
    name: "Simple role check (Pass)",
    query: buildQuery(() => "p1"),
    roles: [
      {
        name: "r1",
        permissions: ["p1", "p2", "p3"],
      },
      { name: "r2", permissions: ["p4", "p5", "p6"] },
    ],
    expected: { status: 200, valid: true },
  },
  {
    name: "Simple role check (Fail)",
    query: buildQuery(() => "p7"),
    roles: [
      {
        name: "r1",
        permissions: ["p1", "p2", "p3", "p4", "p5", "p6"],
      },
    ],
    expected: { status: 200, valid: false },
  },
  {
    name: "'and' of two permissions (Pass)",
    query: buildQuery(({ and }) => and("p1", "p2")),
    roles: [{ name: "r1", permissions: ["p1", "p2", "p3", "p4", "p5", "p6"] }],
    expected: { status: 200, valid: true },
  },
  {
    name: "'and' of two permissions (Fail)",
    query: buildQuery(({ and }) => and("p1", "p7")),
    roles: [{ name: "r1", permissions: ["p1", "p2", "p3", "p4", "p5", "p6"] }],
    expected: { status: 200, valid: false },
  },
  {
    name: "'or' of two permissions (Pass)",
    query: buildQuery(({ or }) => or("p1", "p7")),
    roles: [{ name: "r1", permissions: ["p1", "p2", "p3", "p4", "p5", "p6"] }],
    expected: { status: 200, valid: true },
  },
  {
    name: "or' of two permissions (Fail)",
    query: buildQuery(({ or }) => or("p7", "p3")),
    roles: [
      {
        name: "r1",
        permissions: ["p1"],
      },
      {
        name: "r2",
        permissions: ["p2"],
      },
      {
        name: "r4",
        permissions: ["p4"],
      },
      {
        name: "r5",
        permissions: ["p5"],
      },
      {
        name: "r6",
        permissions: ["p6"],
      },
    ],
    expected: { status: 200, valid: false },
  },
  {
    name: "and' and 'or' combination (Pass)",
    query: buildQuery(({ and, or }) => and("p1", or("p2", "p3"))),
    roles: [{ name: "r1", permissions: ["p1", "p2", "p3", "p4", "p5", "p6"] }],
    expected: { status: 200, valid: true },
  },
  {
    name: "'and' and 'or' combination (Fail)",
    query: buildQuery(({ and, or }) => and("p1", or("p7", "p5"))),
    roles: [
      {
        name: "r1",
        permissions: ["p2", "p3"],
      },
      {
        name: "r2",
        permissions: ["p4"],
      },
      {
        name: "r3",
        permissions: ["p5", "p6"],
      },
    ],
    expected: { status: 200, valid: false },
  },
  {
    name: "Deep nesting of 'and'(Pass)",
    query: buildQuery(({ and }) => and("p1", and("p2", and("p3", "p4")))),
    roles: [{ name: "r1", permissions: ["p1", "p2", "p3", "p4", "p5", "p6"] }],
    expected: { status: 200, valid: true },
  },
  {
    name: "Deep nesting of 'and' (Fail)",
    query: buildQuery(({ and }) => and("p1", and("p7", "p3"))),
    roles: [{ name: "r1", permissions: ["p1", "p2", "p3", "p4", "p5", "p6"] }],
    expected: { status: 200, valid: false },
  },
  {
    name: "Deep nesting of 'or'(Pass)",
    query: buildQuery(({ or }) => or("p1", or("p2", or("p3", "p4")))),
    roles: [{ name: "r1", permissions: ["p1", "p2", "p3", "p4", "p5", "p6"] }],
    expected: { status: 200, valid: true },
  },
  {
    name: "Deep nesting of 'or' (Fail)",
    query: buildQuery(({ or }) => or("p7", or("p5", "p6"))),
    roles: [
      {
        name: "r1",
        permissions: ["p1", "p2", "p3"],
      },
      {
        name: "r2",
        permissions: ["p4"],
      },
    ],
    expected: { status: 200, valid: false },
  },
  {
    name: "Complex combination of 'and' and 'or'(Pass)",
    query: buildQuery(({ and, or }) => or(and("p1", "p2"), and("p3", "p4"))),
    roles: [{ name: "r1", permissions: ["p1", "p2", "p3", "p4", "p5", "p6"] }],
    expected: { status: 200, valid: true },
  },
  {
    name: "Complex combination of 'and' and 'or' (Fail)",
    query: buildQuery(({ and, or }) => or(and("p1", "p7"), and("p5", "p6"))),
    roles: [
      {
        name: "r1",
        permissions: ["p1", "p2", "p3", "p4", "p6"],
      },
    ],
    expected: { status: 200, valid: false },
  },
  {
    name: "Multiple levels of nesting(Pass)",
    query: buildQuery(({ and, or }) => or(and("p1", or("p2", and("p3", "p4"))), "p5")),
    roles: [{ name: "r1", permissions: ["p1", "p2", "p3", "p4", "p5", "p6"] }],
    expected: { status: 200, valid: true },
  },
  {
    name: "Multiple levels of nesting (Fail)",
    query: buildQuery(({ and, or }) => or(and("p1", or("p7", and("p3", "p4"))), "p6")),
    roles: [{ name: "r1", permissions: ["p2", "p3", "p4", "p5"] }],
    expected: { status: 200, valid: false },
  },
  {
    name: "Complex combination of 'and' and 'or' at different levels (Pass)",
    query: buildQuery(({ and, or }) => or(and("p1", or("p2", and("p3", "p4"))), and("p5", "p6"))),
    roles: [{ name: "r1", permissions: ["p1", "p2", "p3", "p4", "p5", "p6"] }],
    expected: { status: 200, valid: true },
  },
  {
    name: "Complex combination of 'and' and 'or' at different levels (Fail)",
    query: buildQuery(({ and, or }) => or(and("p1", or("p7", and("p3", "p4"))), and("p5", "p7"))),
    roles: [{ name: "r1", permissions: ["p2", "p3", "p4", "p5", "p6"] }],
    expected: { status: 200, valid: false },
  },
  // {
  //   name: "Deep nesting of 'and' and 'or'(Pass)",
  //   query: buildQuery(({ and, or }) => and("p1", or("p2", and("p3", or("p4", "p5"))))),
  //   roles: [{ name: "r1", permissions: ["p1", "p2", "p3", "p4", "p5", "p6"] }],
  //   expected: { status: 200, valid: true },
  // },
  // {
  //   name: "Deep nesting of 'and' and 'or' (Fail)",
  //   query: buildQuery(({ and, or }) => and("p1", or("p7", and("p3", or("p4", "p5"))))),
  //   roles: [{ name: "r1", permissions: ["p2", "p3", "p4", "p5", "p6"] }],
  //   expected: { status: 200, valid: false },
  // },
])("$name", async ({ roles, query, expected }) => {
  test(
    `returns valid=${expected.valid}`,
    async (t) => {
      const h = await IntegrationHarness.init(t);
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
      expect(
        res.status,
        `exptected ${expected.status}, received: ${JSON.stringify(res, null, 2)}`,
      ).toEqual(expected.status);
      expect(
        res.body.valid,
        `key is ${res.body.valid ? "valid" : "not valid"}, received body: ${JSON.stringify(
          res.body,
        )}`,
      ).toBe(expected.valid);
    },
    { timeout: 60_000 },
  );
});
