import { rolesQueryPayload } from "@/app/(app)/[workspaceSlug]/authorization/roles/components/table/query-logs.schema";
import type { RolesFilterOperator } from "@/app/(app)/[workspaceSlug]/authorization/roles/filters.schema";
import { db, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";

export const DEFAULT_LIMIT = 50;

export const roleBasic = z.object({
  roleId: z.string(),
  name: z.string(),
  description: z.string(),
  lastUpdated: z.number(),
});

export type RoleBasic = z.infer<typeof roleBasic>;

const rolesResponse = z.object({
  roles: z.array(roleBasic),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.number().int().nullish(),
});

export const queryRoles = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(rolesQueryPayload)
  .output(rolesResponse)
  .query(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;
    const { cursor, name, description, keyName, keyId, permissionSlug, permissionName } = input;

    // Build filter conditions
    const nameFilter = buildFilterConditions(name, "name");
    const descriptionFilter = buildFilterConditions(description, "description");
    const keyFilter = buildKeyFilter(keyName, keyId, workspaceId);
    const permissionFilter = buildPermissionFilter(permissionName, permissionSlug, workspaceId);

    // Get total count first
    const countResult = await db.execute(sql`
      SELECT COUNT(*) as total
      FROM roles
      WHERE workspace_id = ${workspaceId}
        ${nameFilter}
        ${descriptionFilter}
        ${keyFilter}
        ${permissionFilter}
    `);

    const total = (countResult.rows[0] as { total: number }).total;

    if (total === 0) {
      return {
        roles: [],
        hasMore: false,
        total: 0,
        nextCursor: undefined,
      };
    }

    const result = await db.execute(sql`
      SELECT id, name, description, updated_at_m
      FROM roles
      WHERE workspace_id = ${workspaceId}
        ${cursor ? sql`AND updated_at_m < ${cursor}` : sql``}
        ${nameFilter}
        ${descriptionFilter}
        ${keyFilter}
        ${permissionFilter}
      ORDER BY updated_at_m DESC
      LIMIT ${DEFAULT_LIMIT + 1}
    `);

    const rows = result.rows as {
      id: string;
      name: string;
      description: string | null;
      updated_at_m: number;
    }[];

    const hasMore = rows.length > DEFAULT_LIMIT;
    const items = hasMore ? rows.slice(0, -1) : rows;

    const rolesResponseData: RoleBasic[] = items.map((row) => ({
      roleId: row.id,
      name: row.name || "",
      description: row.description || "",
      lastUpdated: Number(row.updated_at_m) || 0,
    }));

    return {
      roles: rolesResponseData,
      hasMore,
      total: Number(total) || 0,
      nextCursor:
        hasMore && items.length > 0
          ? Number(items[items.length - 1].updated_at_m) || undefined
          : undefined,
    };
  });

function buildKeyFilter(
  nameFilters:
    | {
        value: string;
        operator: RolesFilterOperator;
      }[]
    | null
    | undefined,
  idFilters:
    | {
        value: string;
        operator: RolesFilterOperator;
      }[]
    | null
    | undefined,
  workspaceId: string,
) {
  const conditions = [];

  if (nameFilters && nameFilters.length > 0) {
    const nameConditions = nameFilters.map((filter) => {
      const value = filter.value;
      switch (filter.operator) {
        case "is":
          return sql`id IN (
            SELECT DISTINCT kr.role_id
            FROM keys_roles kr
            JOIN \`keys\` k ON kr.key_id = k.id
            WHERE kr.workspace_id = ${workspaceId}
              AND k.name = ${value}
          )`;
        case "contains":
          return sql`id IN (
            SELECT DISTINCT kr.role_id
            FROM keys_roles kr
            JOIN \`keys\` k ON kr.key_id = k.id
            WHERE kr.workspace_id = ${workspaceId}
              AND k.name LIKE ${`%${value}%`}
          )`;
        case "startsWith":
          return sql`id IN (
            SELECT DISTINCT kr.role_id
            FROM keys_roles kr
            JOIN \`keys\` k ON kr.key_id = k.id
            WHERE kr.workspace_id = ${workspaceId}
              AND k.name LIKE ${`${value}%`}
          )`;
        case "endsWith":
          return sql`id IN (
            SELECT DISTINCT kr.role_id
            FROM keys_roles kr
            JOIN \`keys\` k ON kr.key_id = k.id
            WHERE kr.workspace_id = ${workspaceId}
              AND k.name LIKE ${`%${value}`}
          )`;
        default:
          throw new Error(`Invalid operator: ${filter.operator}`);
      }
    });
    conditions.push(sql`(${sql.join(nameConditions, sql` OR `)})`);
  }

  if (idFilters && idFilters.length > 0) {
    const idConditions = idFilters.map((filter) => {
      const value = filter.value;
      switch (filter.operator) {
        case "is":
          return sql`id IN (
            SELECT DISTINCT kr.role_id
            FROM keys_roles kr
            JOIN \`keys\` k ON kr.key_id = k.id
            WHERE kr.workspace_id = ${workspaceId}
              AND k.id = ${value}
          )`;
        case "contains":
          return sql`id IN (
            SELECT DISTINCT kr.role_id
            FROM keys_roles kr
            JOIN \`keys\` k ON kr.key_id = k.id
            WHERE kr.workspace_id = ${workspaceId}
              AND k.id LIKE ${`%${value}%`}
          )`;
        case "startsWith":
          return sql`id IN (
            SELECT DISTINCT kr.role_id
            FROM keys_roles kr
            JOIN \`keys\` k ON kr.key_id = k.id
            WHERE kr.workspace_id = ${workspaceId}
              AND k.id LIKE ${`${value}%`}
          )`;
        case "endsWith":
          return sql`id IN (
            SELECT DISTINCT kr.role_id
            FROM keys_roles kr
            JOIN \`keys\` k ON kr.key_id = k.id
            WHERE kr.workspace_id = ${workspaceId}
              AND k.id LIKE ${`%${value}`}
          )`;
        default:
          throw new Error(`Invalid operator: ${filter.operator}`);
      }
    });
    conditions.push(sql`(${sql.join(idConditions, sql` OR `)})`);
  }

  if (conditions.length === 0) {
    return sql``;
  }

  return sql`AND (${sql.join(conditions, sql` AND `)})`;
}

function buildFilterConditions(
  filters:
    | {
        value: string;
        operator: RolesFilterOperator;
      }[]
    | null
    | undefined,
  columnName: string,
) {
  if (!filters || filters.length === 0) {
    return sql``;
  }

  const conditions = filters.map((filter) => {
    const value = filter.value;
    switch (filter.operator) {
      case "is":
        return sql`${sql.identifier(columnName)} = ${value}`;
      case "contains":
        return sql`${sql.identifier(columnName)} LIKE ${`%${value}%`}`;
      case "startsWith":
        return sql`${sql.identifier(columnName)} LIKE ${`${value}%`}`;
      case "endsWith":
        return sql`${sql.identifier(columnName)} LIKE ${`%${value}`}`;
      default:
        throw new Error(`Invalid operator: ${filter.operator}`);
    }
  });

  return sql`AND (${sql.join(conditions, sql` OR `)})`;
}

function buildPermissionFilter(
  nameFilters:
    | {
        value: string;
        operator: RolesFilterOperator;
      }[]
    | null
    | undefined,
  slugFilters:
    | {
        value: string;
        operator: RolesFilterOperator;
      }[]
    | null
    | undefined,
  workspaceId: string,
) {
  const conditions = [];

  if (nameFilters && nameFilters.length > 0) {
    const nameConditions = nameFilters.map((filter) => {
      const value = filter.value;
      switch (filter.operator) {
        case "is":
          return sql`id IN (
            SELECT DISTINCT rp.role_id
            FROM roles_permissions rp
            JOIN permissions p ON rp.permission_id = p.id
            WHERE rp.workspace_id = ${workspaceId}
              AND p.name = ${value}
          )`;
        case "contains":
          return sql`id IN (
            SELECT DISTINCT rp.role_id
            FROM roles_permissions rp
            JOIN permissions p ON rp.permission_id = p.id
            WHERE rp.workspace_id = ${workspaceId}
              AND p.name LIKE ${`%${value}%`}
          )`;
        case "startsWith":
          return sql`id IN (
            SELECT DISTINCT rp.role_id
            FROM roles_permissions rp
            JOIN permissions p ON rp.permission_id = p.id
            WHERE rp.workspace_id = ${workspaceId}
              AND p.name LIKE ${`${value}%`}
          )`;
        case "endsWith":
          return sql`id IN (
            SELECT DISTINCT rp.role_id
            FROM roles_permissions rp
            JOIN permissions p ON rp.permission_id = p.id
            WHERE rp.workspace_id = ${workspaceId}
              AND p.name LIKE ${`%${value}`}
          )`;
        default:
          throw new Error(`Invalid operator: ${filter.operator}`);
      }
    });
    conditions.push(sql`(${sql.join(nameConditions, sql` OR `)})`);
  }

  if (slugFilters && slugFilters.length > 0) {
    const slugConditions = slugFilters.map((filter) => {
      const value = filter.value;
      switch (filter.operator) {
        case "is":
          return sql`id IN (
            SELECT DISTINCT rp.role_id
            FROM roles_permissions rp
            JOIN permissions p ON rp.permission_id = p.id
            WHERE rp.workspace_id = ${workspaceId}
              AND p.slug = ${value}
          )`;
        case "contains":
          return sql`id IN (
            SELECT DISTINCT rp.role_id
            FROM roles_permissions rp
            JOIN permissions p ON rp.permission_id = p.id
            WHERE rp.workspace_id = ${workspaceId}
              AND p.slug LIKE ${`%${value}%`}
          )`;
        case "startsWith":
          return sql`id IN (
            SELECT DISTINCT rp.role_id
            FROM roles_permissions rp
            JOIN permissions p ON rp.permission_id = p.id
            WHERE rp.workspace_id = ${workspaceId}
              AND p.slug LIKE ${`${value}%`}
          )`;
        case "endsWith":
          return sql`id IN (
            SELECT DISTINCT rp.role_id
            FROM roles_permissions rp
            JOIN permissions p ON rp.permission_id = p.id
            WHERE rp.workspace_id = ${workspaceId}
              AND p.slug LIKE ${`%${value}`}
          )`;
        default:
          throw new Error(`Invalid operator: ${filter.operator}`);
      }
    });
    conditions.push(sql`(${sql.join(slugConditions, sql` OR `)})`);
  }

  if (conditions.length === 0) {
    return sql``;
  }

  return sql`AND (${sql.join(conditions, sql` AND `)})`;
}
