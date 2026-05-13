import type { KeysOverviewFilterUrlValue } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/filters.schema";
import { type InferSelectModel, type SQL, and, db, desc, eq, inArray, isNull, sql } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import {
  identities,
  keys,
  keysPermissions,
  keysRoles,
  permissions,
  roles,
} from "@unkey/db/src/schema";

type BaseKey = InferSelectModel<typeof keys>;
type BaseRole = InferSelectModel<typeof roles>;
type BasePermission = InferSelectModel<typeof permissions>;

type DatabaseKey = Pick<
  BaseKey,
  | "id"
  | "keyAuthId"
  | "name"
  | "ownerId"
  | "identityId"
  | "meta"
  | "enabled"
  | "remaining"
  | "environment"
  | "workspaceId"
> & {
  permissions: {
    permission: Pick<BasePermission, "name" | "description" | "createdAtM" | "updatedAtM">;
  }[];
  roles: {
    role: Pick<BaseRole, "name" | "description" | "createdAtM" | "updatedAtM">;
  }[];
  identity: {
    externalId: string;
  } | null;
};

type KeyDetails = {
  id: DatabaseKey["id"];
  key_auth_id: DatabaseKey["keyAuthId"];
  name: DatabaseKey["name"];
  owner_id: DatabaseKey["ownerId"];
  identity_id: DatabaseKey["identityId"];
  meta: DatabaseKey["meta"];
  enabled: DatabaseKey["enabled"];
  remaining_requests: DatabaseKey["remaining"];
  environment: DatabaseKey["environment"];
  workspace_id: DatabaseKey["workspaceId"];
  identity: { external_id: string } | null;
  roles: {
    name: BaseRole["name"];
    description: BaseRole["description"];
    createdAt?: BaseRole["createdAtM"];
    updatedAt?: BaseRole["updatedAtM"];
  }[];
  permissions: {
    name: BasePermission["name"];
    description: BasePermission["description"];
    createdAt?: BasePermission["createdAtM"];
    updatedAt?: BasePermission["updatedAtM"];
  }[];
};

export interface QueryApiKeysInput {
  apiId: string;
  workspaceId: string;
  keyIds: KeysOverviewFilterUrlValue[] | null;
  names?: KeysOverviewFilterUrlValue[] | null;
  identities?: KeysOverviewFilterUrlValue[] | null;
}

export interface QueryApiKeysResult {
  keyspaceId: string;
  keys: DatabaseKey[];
  keyIds: KeysOverviewFilterUrlValue[] | null;
  // True when more matching keys exist beyond MAX_KEYS_PER_QUERY. Callers can
  // surface this to the user (e.g. "refine your filter") instead of silently
  // displaying a truncated result.
  hasMore: boolean;
}

// Drizzle parameterizes LIKE patterns (no SQL injection risk), but MySQL still
// interprets % and _ as wildcards at query time. A user searching for a key
// named "100%_test" would otherwise match "100xyz_test" etc. Escape the three
// LIKE metacharacters with a non-special character so we can pass ESCAPE '!'
// inline — backslash would need to be doubled in the SQL string literal and is
// fragile across sql_mode settings (e.g. NO_BACKSLASH_ESCAPES).
function escapeLikePattern(value: string): string {
  return value.replace(/[%_!]/g, "!$&");
}

// Cap the rows we ever scan from a single keyspace. The dashboard only renders
// a single page of keys, and ClickHouse always narrows us to a small window
// via keyIds first, so this only ever matters when someone filters by name or
// identity without a keyIds cursor.
const MAX_KEYS_PER_QUERY = 1000;

/**
 * Fetches keys under an API's keyspace that match the given filters, along with
 * each key's roles, permissions, and identity.
 *
 * Built as a flat keys query plus bulk lookups for relations so that MySQL uses
 * `keys.keys_id_unique` / `keys.key_auth_id_deleted_at_idx` directly instead of
 * the nested lateral + JSON aggregation that Drizzle's relational query API
 * generates (which was reading millions of rows per call).
 */
export async function queryApiKeys({
  apiId,
  workspaceId,
  keyIds: keyIdsFromInput,
  names: namesFromInput,
  identities: identitiesFromInput,
}: QueryApiKeysInput): Promise<QueryApiKeysResult> {
  const api = await getApi(apiId, workspaceId);
  if (!api || !api.keyAuth?.id) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "API not found or does not have key authentication enabled",
    });
  }
  const keyspaceId = api.keyAuth.id;

  const conditions: SQL<unknown>[] = [eq(keys.keyAuthId, keyspaceId), isNull(keys.deletedAtM)];

  if (namesFromInput && namesFromInput.length > 0) {
    const nameIsValues = namesFromInput
      .filter((f) => f.operator === "is")
      .map((f) => f.value as string);
    if (nameIsValues.length > 0) {
      conditions.push(inArray(keys.name, nameIsValues));
    }
    for (const f of namesFromInput) {
      const value = f.value;
      if (typeof value !== "string") {
        continue;
      }
      const escaped = escapeLikePattern(value);
      if (f.operator === "contains") {
        conditions.push(sql`${keys.name} LIKE ${`%${escaped}%`} ESCAPE '!'`);
      } else if (f.operator === "startsWith") {
        conditions.push(sql`${keys.name} LIKE ${`${escaped}%`} ESCAPE '!'`);
      } else if (f.operator === "endsWith") {
        conditions.push(sql`${keys.name} LIKE ${`%${escaped}`} ESCAPE '!'`);
      }
    }
  }

  if (keyIdsFromInput && keyIdsFromInput.length > 0) {
    const idIsValues = keyIdsFromInput
      .filter((f) => f.operator === "is")
      .map((f) => f.value as string);
    if (idIsValues.length > 0) {
      conditions.push(inArray(keys.id, idIsValues));
    }
    for (const f of keyIdsFromInput) {
      const value = f.value;
      if (typeof value !== "string") {
        continue;
      }
      if (f.operator === "contains") {
        conditions.push(sql`${keys.id} LIKE ${`%${escapeLikePattern(value)}%`} ESCAPE '!'`);
      }
    }
  }

  if (identitiesFromInput && identitiesFromInput.length > 0) {
    const identityOrs: SQL<unknown>[] = [];
    for (const filter of identitiesFromInput) {
      const value = filter.value;
      if (typeof value !== "string") {
        continue;
      }

      let externalMatch: SQL<unknown>;
      let ownerMatch: SQL<unknown>;
      const escaped = escapeLikePattern(value);
      switch (filter.operator) {
        case "contains":
          externalMatch = sql`${identities.externalId} LIKE ${`%${escaped}%`} ESCAPE '!'`;
          ownerMatch = sql`${keys.ownerId} LIKE ${`%${escaped}%`} ESCAPE '!'`;
          break;
        case "startsWith":
          externalMatch = sql`${identities.externalId} LIKE ${`${escaped}%`} ESCAPE '!'`;
          ownerMatch = sql`${keys.ownerId} LIKE ${`${escaped}%`} ESCAPE '!'`;
          break;
        case "endsWith":
          externalMatch = sql`${identities.externalId} LIKE ${`%${escaped}`} ESCAPE '!'`;
          ownerMatch = sql`${keys.ownerId} LIKE ${`%${escaped}`} ESCAPE '!'`;
          break;
        default:
          externalMatch = sql`${identities.externalId} = ${value}`;
          ownerMatch = sql`${keys.ownerId} = ${value}`;
      }

      identityOrs.push(
        sql`EXISTS (SELECT 1 FROM ${identities} WHERE ${identities.id} = ${keys.identityId} AND ${externalMatch})`,
      );
      identityOrs.push(ownerMatch);
    }

    if (identityOrs.length > 0) {
      conditions.push(sql`(${sql.join(identityOrs, sql` OR `)})`);
    }
  }

  // Fetch one extra row to detect truncation without a separate count query.
  // Deterministic ORDER BY so the cutoff is repeatable and the new
  // (key_auth_id, deleted_at_m, last_used_at) composite serves the sort. `pk`
  // is the tiebreaker — unique, monotonic, and InnoDB stores it as the implicit
  // trailing column of every secondary index so MySQL can avoid a filesort.
  const rawRows = await db
    .select({
      id: keys.id,
      keyAuthId: keys.keyAuthId,
      name: keys.name,
      ownerId: keys.ownerId,
      identityId: keys.identityId,
      meta: keys.meta,
      enabled: keys.enabled,
      remaining: keys.remaining,
      environment: keys.environment,
      workspaceId: keys.workspaceId,
    })
    .from(keys)
    .where(and(...conditions))
    .orderBy(desc(keys.lastUsedAt), desc(keys.pk))
    .limit(MAX_KEYS_PER_QUERY + 1)
    .catch((err) => {
      console.error("Database query error:", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve API information. If this issue persists, please contact support@unkey.com with the time this occurred.",
      });
    });

  const hasMore = rawRows.length > MAX_KEYS_PER_QUERY;
  const keyRows = hasMore ? rawRows.slice(0, MAX_KEYS_PER_QUERY) : rawRows;

  if (hasMore) {
    // Log rather than throw so the dashboard still renders a page — but operators
    // can spot when a workspace regularly hits the cap and needs a refined filter.
    console.warn(
      `queryApiKeys: truncated at ${MAX_KEYS_PER_QUERY} keys for apiId=${apiId} workspaceId=${workspaceId}`,
    );
  }

  if (keyRows.length === 0) {
    return {
      keyspaceId,
      keys: [],
      keyIds: keyIdsFromInput,
      hasMore: false,
    };
  }

  const matchedKeyIds = keyRows.map((k) => k.id);
  const identityIdList = Array.from(
    new Set(
      keyRows
        .map((k) => k.identityId)
        .filter((id): id is string => typeof id === "string" && id.length > 0),
    ),
  );

  const [roleRows, permissionRows, identityRows] = await Promise.all([
    db
      .select({
        keyId: keysRoles.keyId,
        name: roles.name,
        description: roles.description,
        createdAtM: roles.createdAtM,
        updatedAtM: roles.updatedAtM,
      })
      .from(keysRoles)
      .innerJoin(roles, eq(keysRoles.roleId, roles.id))
      .where(inArray(keysRoles.keyId, matchedKeyIds)),
    db
      .select({
        keyId: keysPermissions.keyId,
        name: permissions.name,
        description: permissions.description,
        createdAtM: permissions.createdAtM,
        updatedAtM: permissions.updatedAtM,
      })
      .from(keysPermissions)
      .innerJoin(permissions, eq(keysPermissions.permissionId, permissions.id))
      .where(inArray(keysPermissions.keyId, matchedKeyIds)),
    identityIdList.length > 0
      ? db
          .select({ id: identities.id, externalId: identities.externalId })
          .from(identities)
          .where(inArray(identities.id, identityIdList))
      : Promise.resolve([] as { id: string; externalId: string }[]),
  ]).catch((err) => {
    console.error("Database query error (key relation lookups):", err);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message:
        "Failed to retrieve API information. If this issue persists, please contact support@unkey.com with the time this occurred.",
    });
  });

  const rolesByKey = new Map<string, DatabaseKey["roles"]>();
  for (const r of roleRows) {
    const entry = rolesByKey.get(r.keyId);
    const rel = {
      role: {
        name: r.name,
        description: r.description,
        createdAtM: r.createdAtM,
        updatedAtM: r.updatedAtM,
      },
    };
    if (entry) {
      entry.push(rel);
    } else {
      rolesByKey.set(r.keyId, [rel]);
    }
  }

  const permissionsByKey = new Map<string, DatabaseKey["permissions"]>();
  for (const p of permissionRows) {
    const entry = permissionsByKey.get(p.keyId);
    const rel = {
      permission: {
        name: p.name,
        description: p.description,
        createdAtM: p.createdAtM,
        updatedAtM: p.updatedAtM,
      },
    };
    if (entry) {
      entry.push(rel);
    } else {
      permissionsByKey.set(p.keyId, [rel]);
    }
  }

  const identityById = new Map(identityRows.map((i) => [i.id, i.externalId]));

  const keysResult: DatabaseKey[] = keyRows.map((k) => ({
    id: k.id,
    keyAuthId: k.keyAuthId,
    name: k.name,
    ownerId: k.ownerId,
    identityId: k.identityId,
    meta: k.meta,
    enabled: k.enabled,
    remaining: k.remaining,
    environment: k.environment,
    workspaceId: k.workspaceId,
    roles: rolesByKey.get(k.id) ?? [],
    permissions: permissionsByKey.get(k.id) ?? [],
    identity:
      k.identityId && identityById.has(k.identityId)
        ? { externalId: identityById.get(k.identityId) as string }
        : null,
  }));

  let keyIdsFilter = keyIdsFromInput;
  if (!keyIdsFilter || keyIdsFilter.length === 0) {
    if (matchedKeyIds.length > 0) {
      keyIdsFilter = matchedKeyIds.map((keyId) => ({
        operator: "is" as const,
        value: keyId,
      }));
    }
  }

  return {
    keyspaceId,
    keys: keysResult,
    keyIds: keyIdsFilter,
    hasMore,
  };
}

export function createKeyDetailsMap(keys: DatabaseKey[]): Map<string, KeyDetails> {
  const keyDetailsMap = new Map<string, KeyDetails>();
  for (const key of keys) {
    const rolesData = key.roles
      ? key.roles
          .filter((roleRelation) => roleRelation.role != null)
          .map((roleRelation) => ({
            name: roleRelation.role.name,
            description: roleRelation.role.description,
            createdAt: roleRelation.role.createdAtM,
            updatedAt: roleRelation.role.updatedAtM,
          }))
      : [];

    const permissionsData = key.permissions
      ? key.permissions
          .filter((permRelation) => permRelation.permission != null)
          .map((permRelation) => ({
            name: permRelation.permission.name,
            description: permRelation.permission.description,
            createdAt: permRelation.permission.createdAtM,
            updatedAt: permRelation.permission.updatedAtM,
          }))
      : [];

    const identityData = key.identity
      ? {
          external_id: key.identity.externalId,
        }
      : null;

    const keyDetails: KeyDetails = {
      id: key.id,
      key_auth_id: key.keyAuthId,
      name: key.name,
      owner_id: key.ownerId,
      identity_id: key.identityId,
      meta: key.meta,
      enabled: key.enabled,
      remaining_requests: key.remaining,
      environment: key.environment,
      workspace_id: key.workspaceId,
      identity: identityData,
      roles: rolesData,
      permissions: permissionsData,
    };
    keyDetailsMap.set(key.id, keyDetails);
  }
  return keyDetailsMap;
}

export function extractRolesAndPermissions(key: DatabaseKey) {
  const roles = key.roles
    ? key.roles
        .filter((roleRelation) => roleRelation.role != null)
        .map((roleRelation) => ({
          name: roleRelation.role.name,
          description: roleRelation.role.description,
        }))
    : [];
  const permissions = key.permissions
    ? key.permissions
        .filter((permRelation) => permRelation.permission != null)
        .map((permRelation) => ({
          name: permRelation.permission.name,
          description: permRelation.permission.description,
        }))
    : [];
  return {
    roles,
    permissions,
  };
}

export const getApi = async (apiId: string, workspaceId: string) => {
  const api = await db.query.apis
    .findFirst({
      where: (api, { and, eq, isNull }) =>
        and(eq(api.id, apiId), eq(api.workspaceId, workspaceId), isNull(api.deletedAtM)),
      with: {
        keyAuth: {
          columns: {
            id: true,
          },
        },
      },
    })
    .catch((err) => {
      console.error("Database query error:", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve API information.",
      });
    });

  return api;
};
