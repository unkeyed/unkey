import { rolesQueryPayload } from "@/app/(app)/authorization/roles/components/table/query-logs.schema";
import type { RolesFilterOperator } from "@/app/(app)/authorization/roles/filters.schema";
import { db, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";

const MAX_ITEMS_TO_SHOW = 3;
const ITEM_SEPARATOR = "|||";
export const DEFAULT_LIMIT = 50;

export const roles = z.object({
  roleId: z.string(),
  name: z.string(),
  description: z.string(),
  lastUpdated: z.number(),
  assignedKeys: z.object({
    items: z.array(z.string()),
    totalCount: z.number().optional(),
  }),
  permissions: z.object({
    items: z.array(z.string()),
    totalCount: z.number().optional(),
  }),
});

export type Roles = z.infer<typeof roles>;

const rolesResponse = z.object({
  roles: z.array(roles),
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

    // Build filter conditions for total count
    const keyFilterForCount = buildKeyFilter(keyName, keyId, workspaceId);
    const permissionFilterForCount = buildPermissionFilter(
      permissionName,
      permissionSlug,
      workspaceId,
    );

    const result = await db.execute(sql`
 SELECT 
   r.id,
   r.name,
   r.description,
   r.updated_at_m,
   
   -- Keys: get first 3 unique names
   (
     SELECT GROUP_CONCAT(sub.display_name ORDER BY sub.sort_key SEPARATOR ${ITEM_SEPARATOR})
     FROM (
       SELECT DISTINCT 
         CASE 
           WHEN k.name IS NULL OR k.name = '' THEN 
             CONCAT(SUBSTRING(k.id, 1, 8), '...', RIGHT(k.id, 4))
           ELSE k.name 
         END as display_name,
         COALESCE(k.name, k.id) as sort_key
       FROM keys_roles kr
       JOIN \`keys\` k ON kr.key_id = k.id
       WHERE kr.role_id = r.id 
         AND kr.workspace_id = ${workspaceId}
       ORDER BY sort_key
       LIMIT ${MAX_ITEMS_TO_SHOW}
     ) sub
   ) as key_items,
   
   -- Keys: total count
   (
     SELECT COUNT(DISTINCT kr.key_id)
     FROM keys_roles kr
     JOIN \`keys\` k ON kr.key_id = k.id
     WHERE kr.role_id = r.id 
       AND kr.workspace_id = ${workspaceId}
   ) as total_keys,
   
   -- Permissions: get first 3 unique names
   (
     SELECT GROUP_CONCAT(sub.name ORDER BY sub.name SEPARATOR ${ITEM_SEPARATOR})
     FROM (
       SELECT DISTINCT p.name
       FROM roles_permissions rp
       JOIN permissions p ON rp.permission_id = p.id
       WHERE rp.role_id = r.id 
         AND rp.workspace_id = ${workspaceId}
         AND p.name IS NOT NULL
       ORDER BY p.name
       LIMIT ${MAX_ITEMS_TO_SHOW}
     ) sub
   ) as permission_items,
   
   -- Permissions: total count
   (
     SELECT COUNT(DISTINCT rp.permission_id)
     FROM roles_permissions rp
     JOIN permissions p ON rp.permission_id = p.id
     WHERE rp.role_id = r.id 
       AND rp.workspace_id = ${workspaceId}
       AND p.name IS NOT NULL
   ) as total_permissions,
   
   -- Total count of roles (with filters applied)
   (
     SELECT COUNT(*) 
     FROM roles 
     WHERE workspace_id = ${workspaceId}
       ${nameFilter}
       ${descriptionFilter}
       ${keyFilterForCount}
       ${permissionFilterForCount}
   ) as grand_total
   
 FROM (
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
 ) r
 ORDER BY r.updated_at_m DESC
`);

    const rows = result.rows as {
      id: string;
      name: string;
      description: string | null;
      updated_at_m: number;
      key_items: string | null;
      total_keys: number;
      permission_items: string | null;
      total_permissions: number;
      grand_total: number;
    }[];

    if (rows.length === 0) {
      return {
        roles: [],
        hasMore: false,
        total: 0,
        nextCursor: undefined,
      };
    }

    const total = rows[0].grand_total;
    const hasMore = rows.length > DEFAULT_LIMIT;
    const items = hasMore ? rows.slice(0, -1) : rows;

    const rolesResponseData: Roles[] = items.map((row) => {
      // Parse concatenated strings back to arrays
      const keyItems = row.key_items
        ? row.key_items.split(ITEM_SEPARATOR).filter((item) => item.trim() !== "")
        : [];
      const permissionItems = row.permission_items
        ? row.permission_items.split(ITEM_SEPARATOR).filter((item) => item.trim() !== "")
        : [];

      return {
        roleId: row.id,
        name: row.name || "",
        description: row.description || "",
        lastUpdated: Number(row.updated_at_m) || 0,
        assignedKeys:
          row.total_keys <= MAX_ITEMS_TO_SHOW
            ? { items: keyItems }
            : { items: keyItems, totalCount: Number(row.total_keys) },
        permissions:
          row.total_permissions <= MAX_ITEMS_TO_SHOW
            ? { items: permissionItems }
            : {
                items: permissionItems,
                totalCount: Number(row.total_permissions),
              },
      };
    });

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

  // Handle name filters
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

  // Handle ID filters
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

  // Join name and ID conditions with AND
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

  // Combine conditions with OR
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

  // Handle name filters
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

  // Handle slug filters
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

  // Join name and slug conditions with AND
  return sql`AND (${sql.join(conditions, sql` AND `)})`;
}
