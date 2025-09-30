import { permissionsQueryPayload } from "@/app/(app)/[workspaceSlug]/authorization/permissions/components/table/query-logs.schema";
import type { PermissionsFilterOperator } from "@/app/(app)/[workspaceSlug]/authorization/permissions/filters.schema";
import { db, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";

export const DEFAULT_LIMIT = 50;

export const permissions = z.object({
  permissionId: z.string(),
  name: z.string(),
  description: z.string(),
  slug: z.string(),
  lastUpdated: z.number(),
  totalConnectedKeys: z.number(),
  totalConnectedRoles: z.number(),
});

export type Permission = z.infer<typeof permissions>;

const permissionsResponse = z.object({
  permissions: z.array(permissions),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.number().int().nullish(),
});

export const queryPermissions = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(permissionsQueryPayload)
  .output(permissionsResponse)
  .query(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;
    const { cursor, name, description, slug, roleName, roleId } = input;

    // Build filter conditions
    const nameFilter = buildFilterConditions(name, "name");
    const descriptionFilter = buildFilterConditions(description, "description");
    const slugFilter = buildFilterConditions(slug, "slug");
    const roleFilter = buildRoleFilter(roleName, roleId, workspaceId);

    // Build filter conditions for total count
    const roleFilterForCount = buildRoleFilter(roleName, roleId, workspaceId);

    const result = await db.execute(sql`
    SELECT
      p.id,
      p.name,
      p.description,
      p.slug,
      p.updated_at_m,

      -- Roles: total count
      (
        SELECT COUNT(DISTINCT rp.role_id)
        FROM roles_permissions rp
        WHERE rp.permission_id = p.id
          AND rp.workspace_id = ${workspaceId}
      ) as total_roles,

      -- Total connected keys through roles
      (
        SELECT COUNT(DISTINCT kr.key_id)
        FROM roles_permissions rp
        INNER JOIN keys_roles kr ON kr.role_id = rp.role_id
        WHERE rp.permission_id = p.id
          AND rp.workspace_id = ${workspaceId}
      ) as total_connected_keys,

      -- Total count of permissions (with filters applied)
      (
        SELECT COUNT(*)
        FROM permissions
        WHERE workspace_id = ${workspaceId}
          ${nameFilter}
          ${descriptionFilter}
          ${slugFilter}
          ${roleFilterForCount}
      ) as grand_total

    FROM (
      SELECT id, name, description, slug, updated_at_m
      FROM permissions
      WHERE workspace_id = ${workspaceId}
        ${cursor ? sql`AND updated_at_m < ${cursor}` : sql``}
        ${nameFilter}
        ${descriptionFilter}
        ${slugFilter}
        ${roleFilter}
      ORDER BY updated_at_m DESC
      LIMIT ${DEFAULT_LIMIT + 1}
    ) p
    ORDER BY p.updated_at_m DESC
`);

    const rows = result.rows as {
      id: string;
      name: string;
      description: string | null;
      slug: string;
      updated_at_m: number;
      total_roles: number;
      total_connected_keys: number;
      grand_total: number;
    }[];

    if (rows.length === 0) {
      return {
        permissions: [],
        hasMore: false,
        total: 0,
        nextCursor: undefined,
      };
    }

    const total = rows[0].grand_total;
    const hasMore = rows.length > DEFAULT_LIMIT;
    const items = hasMore ? rows.slice(0, -1) : rows;

    const permissionsResponseData: Permission[] = items.map((row) => {
      return {
        permissionId: row.id,
        name: row.name || "",
        description: row.description || "",
        slug: row.slug || "",
        lastUpdated: Number(row.updated_at_m) || 0,
        totalConnectedRoles: Number(row.total_roles),
        totalConnectedKeys: Number(row.total_connected_keys) || 0,
      };
    });

    return {
      permissions: permissionsResponseData,
      hasMore,
      total: Number(total) || 0,
      nextCursor:
        hasMore && items.length > 0
          ? Number(items[items.length - 1].updated_at_m) || undefined
          : undefined,
    };
  });

function buildRoleFilter(
  nameFilters:
    | {
        value: string;
        operator: PermissionsFilterOperator;
      }[]
    | null
    | undefined,
  idFilters:
    | {
        value: string;
        operator: PermissionsFilterOperator;
      }[]
    | null
    | undefined,
  workspaceId: string,
) {
  const conditions = [];

  // Handle name filters
  if (nameFilters && nameFilters.length > 0) {
    const nameConditions = nameFilters.map((filter) => {
      const value = filter.value;
      switch (filter.operator) {
        case "is":
          return sql`id IN (
            SELECT DISTINCT rp.permission_id
            FROM roles_permissions rp
            JOIN roles r ON rp.role_id = r.id
            WHERE rp.workspace_id = ${workspaceId}
              AND r.name = ${value}
          )`;
        case "contains":
          return sql`id IN (
            SELECT DISTINCT rp.permission_id
            FROM roles_permissions rp
            JOIN roles r ON rp.role_id = r.id
            WHERE rp.workspace_id = ${workspaceId}
              AND r.name LIKE ${`%${value}%`}
          )`;
        case "startsWith":
          return sql`id IN (
            SELECT DISTINCT rp.permission_id
            FROM roles_permissions rp
            JOIN roles r ON rp.role_id = r.id
            WHERE rp.workspace_id = ${workspaceId}
              AND r.name LIKE ${`${value}%`}
          )`;
        case "endsWith":
          return sql`id IN (
            SELECT DISTINCT rp.permission_id
            FROM roles_permissions rp
            JOIN roles r ON rp.role_id = r.id
            WHERE rp.workspace_id = ${workspaceId}
              AND r.name LIKE ${`%${value}`}
          )`;
        default:
          throw new Error(`Invalid operator: ${filter.operator}`);
      }
    });
    conditions.push(sql`(${sql.join(nameConditions, sql` OR `)})`);
  }

  // Handle ID filters
  if (idFilters && idFilters.length > 0) {
    const idConditions = idFilters.map((filter) => {
      const value = filter.value;
      switch (filter.operator) {
        case "is":
          return sql`id IN (
            SELECT DISTINCT rp.permission_id
            FROM roles_permissions rp
            JOIN roles r ON rp.role_id = r.id
            WHERE rp.workspace_id = ${workspaceId}
              AND r.id = ${value}
          )`;
        case "contains":
          return sql`id IN (
            SELECT DISTINCT rp.permission_id
            FROM roles_permissions rp
            JOIN roles r ON rp.role_id = r.id
            WHERE rp.workspace_id = ${workspaceId}
              AND r.id LIKE ${`%${value}%`}
          )`;
        case "startsWith":
          return sql`id IN (
            SELECT DISTINCT rp.permission_id
            FROM roles_permissions rp
            JOIN roles r ON rp.role_id = r.id
            WHERE rp.workspace_id = ${workspaceId}
              AND r.id LIKE ${`${value}%`}
          )`;
        case "endsWith":
          return sql`id IN (
            SELECT DISTINCT rp.permission_id
            FROM roles_permissions rp
            JOIN roles r ON rp.role_id = r.id
            WHERE rp.workspace_id = ${workspaceId}
              AND r.id LIKE ${`%${value}`}
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

  // Join name and ID conditions with AND
  return sql`AND (${sql.join(conditions, sql` AND `)})`;
}

function buildFilterConditions(
  filters:
    | {
        value: string;
        operator: PermissionsFilterOperator;
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

  // Combine conditions with OR
  return sql`AND (${sql.join(conditions, sql` OR `)})`;
}
