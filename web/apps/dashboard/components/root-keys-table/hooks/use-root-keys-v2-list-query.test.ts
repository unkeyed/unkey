import type { V2RootKey } from "@/lib/root-keys-api";
import { describe, expect, it } from "vitest";
import { selectRootKeysPage } from "./use-root-keys-v2-list-query";

const rootKeys: V2RootKey[] = [
  {
    keyId: "key_a",
    name: "Production",
    start: "unkey_prod",
    end: "aaaa",
    enabled: true,
    createdAt: 100,
    lastUsedAt: 300,
    expires: null,
    permissions: ["unkey:v1:ws_1:apis/*#read_api"],
  },
  {
    keyId: "key_b",
    name: "CI",
    start: "unkey_ci",
    end: "bbbb",
    enabled: true,
    createdAt: 200,
    lastUsedAt: 0,
    expires: null,
    permissions: ["unkey:v1:ws_1:projects/*#deploy_project"],
  },
];

describe("selectRootKeysPage", () => {
  it("filters URN permissions before sorting and paginating", () => {
    expect(
      selectRootKeysPage(rootKeys, {
        page: 1,
        limit: 1,
        sortBy: "createdAt",
        sortOrder: "asc",
        name: null,
        start: null,
        permission: [{ operator: "contains", value: "deploy_project" }],
      }),
    ).toMatchObject({
      total: 1,
      keys: [
        {
          id: "key_b",
          permissions: [
            {
              id: "unkey:v1:ws_1:projects/*#deploy_project",
              name: "unkey:v1:ws_1:projects/*#deploy_project",
            },
          ],
        },
      ],
    });
  });

  it("supports exact name filters and descending last-used sorting", () => {
    const selected = selectRootKeysPage(rootKeys, {
      page: 1,
      limit: 50,
      sortBy: "lastUsedAt",
      sortOrder: "desc",
      name: [{ operator: "is", value: "Production" }],
      start: null,
      permission: null,
    });

    expect(selected.keys.map((key) => key.id)).toEqual(["key_a"]);
  });
});
