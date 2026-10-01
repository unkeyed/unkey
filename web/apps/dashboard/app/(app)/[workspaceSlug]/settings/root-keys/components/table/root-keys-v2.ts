import type { RootKeysSortField } from "@/components/root-keys-table/schema/query-logs.schema";
import type { V2RootKey } from "@/lib/root-keys-api";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";

type NameFilter = { operator: string; value: string };

type RootKeysV2ListParams = {
  page: number;
  limit: number;
  sortBy: RootKeysSortField;
  sortOrder: "asc" | "desc";
  name?: NameFilter[] | null;
};

function matchesName(name: string | null, filters: NameFilter[] | null | undefined) {
  if (!filters?.length) {
    return true;
  }
  const normalized = (name ?? "").toLowerCase();
  return filters.some((filter) => {
    const expected = filter.value.toLowerCase();
    return filter.operator === "is" ? normalized === expected : normalized.includes(expected);
  });
}

function toRootKey(rootKey: V2RootKey): RootKey {
  const permissions = rootKey.permissions.map((permission) => ({
    id: permission,
    name: permission,
  }));

  return {
    id: rootKey.keyId,
    start: rootKey.start,
    createdAt: rootKey.createdAt,
    lastUpdatedAt: null,
    expires: rootKey.expires,
    name: rootKey.name,
    permissionSummary: {
      total: permissions.length,
      categories: {},
      hasCriticalPerm: permissions.some(({ name }) =>
        ["delete", "decrypt", "remove"].some((action) => name.toLowerCase().includes(action)),
      ),
    },
    permissions,
  };
}

export function selectRootKeysPage(
  rootKeys: V2RootKey[],
  params: RootKeysV2ListParams,
): { keys: RootKey[]; total: number } {
  const filtered = rootKeys
    .filter((rootKey) => matchesName(rootKey.name, params.name))
    .map(toRootKey);

  filtered.sort((left, right) => {
    const leftValue = left[params.sortBy] ?? "";
    const rightValue = right[params.sortBy] ?? "";
    const comparison =
      typeof leftValue === "number" && typeof rightValue === "number"
        ? leftValue - rightValue
        : String(leftValue).localeCompare(String(rightValue));
    return (comparison || left.id.localeCompare(right.id)) * (params.sortOrder === "asc" ? 1 : -1);
  });

  const offset = (params.page - 1) * params.limit;
  return {
    keys: filtered.slice(offset, offset + params.limit),
    total: filtered.length,
  };
}
