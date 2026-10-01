import type { V2RootKey } from "@/lib/root-keys-api";
import { describe, expect, it } from "vitest";
import { selectRootKeysPage } from "./root-keys-v2";

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
  it("filters before sorting and paginating", () => {
    expect(
      selectRootKeysPage(rootKeys, {
        page: 1,
        limit: 1,
        sortBy: "createdAt",
        sortOrder: "asc",
        name: [{ operator: "contains", value: "ci" }],
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

  it("uses the key id as a stable tie breaker", () => {
    const selected = selectRootKeysPage(
      rootKeys.map((rootKey) => ({ ...rootKey, name: "Same name" })),
      {
        page: 1,
        limit: 50,
        sortBy: "name",
        sortOrder: "desc",
        name: null,
      },
    );

    expect(selected.keys.map((key) => key.id)).toEqual(["key_b", "key_a"]);
  });
});
