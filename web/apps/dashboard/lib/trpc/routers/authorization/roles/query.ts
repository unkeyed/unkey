import type { RolesFilterOperator } from "@/app/(app)/[workspaceSlug]/authorization/roles/filters.schema";
import { rolesQueryPayload } from "@/components/roles-table/schema/roles.schema";
import { db, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

export const DEFAULT_LIMIT = 50;

export const roleBasic = z.object({
  roleId: z.string(),
  name: z.string(),
  description: z.string(),
  lastUpdated: z.number(),
  assignedKeys: z.number(),
  assignedPermissions: z.number(),
});

export type RoleBasic = z.infer<typeof roleBasic>;

const rolesResponse = z.object({
  roles: z.array(roleBasic),
  total: z.number(),
});

// Maps sortBy field names to SQL column references
const SORT_FIELD_TO_INNER_SQL: Record<string, string> = {
  name: "name",
  lastUpdated: "updated_at_m",
};
const SORT_FIELD_TO_OUTER_SQL: Record<string, string> = {
  name: "r.name",
  lastUpdated: "r.updated_at_m",
  assignedKeys: "assigned_keys",
  assignedPermissions: "assigned_permissions",
};

const COMPUTED_SORT_FIELDS = new Set(["assignedKeys", "assignedPermissions"]);

export const queryRoles = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(rolesQueryPayload)
  .output(rolesResponse)
  .query(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;
    const page = input.page ?? 1;
    const pageSize = input.limit ?? DEFAULT_LIMIT;
    const sortBy = input.sortBy ?? "lastUpdated";
    const sortOrder = input.sortOrder ?? "desc";
    const { name, description, keyName, keyId, permissionSlug, permissionName } = input;

    const offset = (page - 1) * pageSize;

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

    const total = (countResult[0] as unknown as { total: number }[])[0].total;

    if (total === 0) {
      return {
        roles: [],
        total: 0,
      };
    }

    const direction = sortOrder === "asc" ? sql`ASC` : sql`DESC`;
    const isComputedSort = COMPUTED_SORT_FIELDS.has(sortBy);

    // For direct field sorts, apply LIMIT/OFFSET in the inner query.
    // For computed field sorts, we must apply LIMIT/OFFSET in the outer query
    // because the computed values aren't available in the inner query.
    const innerOrderColumn = sql.raw(SORT_FIELD_TO_INNER_SQL[sortBy] ?? "updated_at_m");
    const outerOrderColumn = sql.raw(SORT_FIELD_TO_OUTER_SQL[sortBy] ?? "r.updated_at_m");

    // Break ties on `id` so rows with equal sort keys (e.g. same updated_at_m)
    // don't shuffle between pages.
    const result = await db.execute(sql`
      SELECT
        r.id,
        r.name,
        r.description,
        r.updated_at_m,
        COALESCE(kr_count.total_keys, 0) AS assigned_keys,
        COALESCE(rp_count.total_permissions, 0) AS assigned_permissions
      FROM (
        SELECT id, name, description, updated_at_m
        FROM roles
        WHERE workspace_id = ${workspaceId}
          ${nameFilter}
          ${descriptionFilter}
          ${keyFilter}
          ${permissionFilter}
        ORDER BY ${innerOrderColumn} ${direction}, id ${direction}
        ${isComputedSort ? sql`` : sql`LIMIT ${pageSize} OFFSET ${offset}`}
      ) r
      LEFT JOIN (
        SELECT role_id, COUNT(*) AS total_keys
        FROM keys_roles
        WHERE workspace_id = ${workspaceId}
        GROUP BY role_id
      ) kr_count ON kr_count.role_id = r.id
      LEFT JOIN (
        SELECT role_id, COUNT(*) AS total_permissions
        FROM roles_permissions
        WHERE workspace_id = ${workspaceId}
        GROUP BY role_id
      ) rp_count ON rp_count.role_id = r.id
      ORDER BY ${outerOrderColumn} ${direction}, r.id ${direction}
      ${isComputedSort ? sql`LIMIT ${pageSize} OFFSET ${offset}` : sql``}
    `);

    const rows = result[0] as unknown as {
      id: string;
      name: string;
      description: string | null;
      updated_at_m: number;
      assigned_keys: number;
      assigned_permissions: number;
    }[];

    const rolesResponseData: RoleBasic[] = rows.map((row) => ({
      roleId: row.id,
      name: row.name || "",
      description: row.description || "",
      lastUpdated: Number(row.updated_at_m) || 0,
      assignedKeys: Number(row.assigned_keys) || 0,
      assignedPermissions: Number(row.assigned_permissions) || 0,
    }));

    return {
      roles: rolesResponseData,
      total: Number(total) || 0,
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
