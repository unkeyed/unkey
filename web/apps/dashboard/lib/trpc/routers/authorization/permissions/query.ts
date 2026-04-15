import type { PermissionsFilterOperator } from "@/app/(app)/[workspaceSlug]/authorization/permissions/filters.schema";
import {
  type PermissionsSortField,
  type PermissionsSortOrder,
  permissionsQueryPayload,
} from "@/components/permissions-table/schema/permissions.schema";
import { db, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
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
  total: z.number(),
});

// Maps client sort field names to SQL expressions used in ORDER BY.
// "totalConnectedRoles" and "totalConnectedKeys" are computed columns, so they
// must be sorted in the outer query using their aliases.
const SORT_FIELD_TO_INNER_SQL: Record<string, string> = {
  name: "name",
  slug: "slug",
  lastUpdated: "updated_at_m",
};

const SORT_FIELD_TO_OUTER_SQL: Record<string, string> = {
  name: "p.name",
  slug: "p.slug",
  lastUpdated: "p.updated_at_m",
  totalConnectedRoles: "total_roles",
  totalConnectedKeys: "total_connected_keys",
};

function buildOrderBy(
  sortBy: PermissionsSortField,
  sortOrder: PermissionsSortOrder,
  context: "inner" | "outer",
) {
  const map = context === "inner" ? SORT_FIELD_TO_INNER_SQL : SORT_FIELD_TO_OUTER_SQL;
  const column = map[sortBy];

  // For computed columns (roles/keys counts), we can't sort in the inner subquery
  // because the values aren't available there. Fall back to updated_at_m.
  const innerColumn = column ?? "updated_at_m";
  const direction = sortOrder === "asc" ? sql`ASC` : sql`DESC`;

  const primaryCol = context === "inner" ? innerColumn : (column ?? "p.updated_at_m");
  const tiebreaker = context === "inner" ? "id" : "p.id";

  return sql`ORDER BY ${sql.raw(primaryCol)} ${direction}, ${sql.raw(tiebreaker)} ${direction}`;
}

export const queryPermissions = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(permissionsQueryPayload)
  .output(permissionsResponse)
  .query(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;
    const { page, limit, name, description, slug, roleName, roleId, sortBy, sortOrder } = input;

    const pageSize = limit ?? DEFAULT_LIMIT;
    const offset = ((page ?? 1) - 1) * pageSize;

    // Build filter conditions
    const nameFilter = buildFilterConditions(name, "name");
    const descriptionFilter = buildFilterConditions(description, "description");
    const slugFilter = buildFilterConditions(slug, "slug");
    const roleFilter = buildRoleFilter(roleName, roleId, workspaceId);

    // Build filter conditions for total count
    const roleFilterForCount = buildRoleFilter(roleName, roleId, workspaceId);

    // For computed columns (totalConnectedRoles, totalConnectedKeys) we need to
    // sort in the outer query, so fetch all filtered rows in the inner query and
    // apply LIMIT/OFFSET in the outer query instead.
    const isComputedSort = sortBy === "totalConnectedRoles" || sortBy === "totalConnectedKeys";

    const innerOrderBy = buildOrderBy(sortBy, sortOrder, "inner");
    const outerOrderBy = buildOrderBy(sortBy, sortOrder, "outer");

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
        ${nameFilter}
        ${descriptionFilter}
        ${slugFilter}
        ${roleFilter}
      ${innerOrderBy}
      ${isComputedSort ? sql`` : sql`LIMIT ${pageSize} OFFSET ${offset}`}
    ) p
    ${outerOrderBy}
    ${isComputedSort ? sql`LIMIT ${pageSize} OFFSET ${offset}` : sql``}
`);

    const rows = result[0] as unknown as {
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
      // When LIMIT/OFFSET yields no rows the per-row grand_total is unavailable.
      // Run a standalone count so pagination still reports the real total.
      const countResult = await db.execute(sql`
        SELECT COUNT(*) as total
        FROM permissions
        WHERE workspace_id = ${workspaceId}
          ${nameFilter}
          ${descriptionFilter}
          ${slugFilter}
          ${roleFilterForCount}
      `);
      const countRows = countResult[0] as unknown as { total: number }[];
      const fallbackTotal = countRows.length > 0 ? Number(countRows[0].total) : 0;

      return {
        permissions: [],
        total: fallbackTotal,
      };
    }

    const total = rows[0].grand_total;

    const permissionsResponseData: Permission[] = rows.map((row) => {
      return {
        permissionId: row.id,
        name: row.name || "",
        description: row.description || "",
        slug: row.slug || "",
        lastUpdated: Number(row.updated_at_m) || 0,
        totalConnectedRoles: Number(row.total_roles) || 0,
        totalConnectedKeys: Number(row.total_connected_keys) || 0,
      };
    });

    return {
      permissions: permissionsResponseData,
      total: Number(total) || 0,
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
