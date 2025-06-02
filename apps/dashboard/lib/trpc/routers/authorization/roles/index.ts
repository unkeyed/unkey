import { db, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";

const MAX_ITEMS_TO_SHOW = 3;
const ITEM_SEPARATOR = "|||";
export const DEFAULT_LIMIT = 50;
const MIN_LIMIT = 1;

const rolesQueryInput = z.object({
  limit: z.number().int().min(MIN_LIMIT).max(DEFAULT_LIMIT).default(DEFAULT_LIMIT),
  cursor: z.number().int().optional(),
});

export const roles = z.object({
  roleId: z.string(),
  slug: z.string(),
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
  .input(rolesQueryInput)
  .output(rolesResponse)
  .query(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;
    const { limit, cursor } = input;

    const result = await db.execute(sql`
      SELECT 
        r.id,
        r.name,
        r.human_readable,
        r.description,
        r.updated_at_m,
        
        -- Keys data (only first ${MAX_ITEMS_TO_SHOW} names)
        GROUP_CONCAT(
          CASE 
            WHEN key_data.key_row_num <= ${MAX_ITEMS_TO_SHOW}
            THEN key_data.key_name 
          END 
          ORDER BY key_data.key_name 
          SEPARATOR ${ITEM_SEPARATOR}
        ) as key_items,
        COALESCE(MAX(key_data.total_keys), 0) as total_keys,
        
        -- Permissions data (only first ${MAX_ITEMS_TO_SHOW} names)
        GROUP_CONCAT(
          CASE 
            WHEN perm_data.perm_row_num <= ${MAX_ITEMS_TO_SHOW}
            THEN perm_data.permission_name 
          END 
          ORDER BY perm_data.permission_name 
          SEPARATOR ${ITEM_SEPARATOR}
        ) as permission_items,
        COALESCE(MAX(perm_data.total_permissions), 0) as total_permissions,
        
        -- Total count
        (SELECT COUNT(*) FROM roles WHERE workspace_id = ${workspaceId}) as grand_total
        
      FROM (
        SELECT *
        FROM roles 
        WHERE workspace_id = ${workspaceId}
          ${cursor ? sql`AND updated_at_m < ${cursor}` : sql``}
        ORDER BY updated_at_m DESC
        LIMIT ${limit + 1}
      ) r
      LEFT JOIN (
        SELECT 
          kr.role_id,
          k.name as key_name,
          ROW_NUMBER() OVER (PARTITION BY kr.role_id ORDER BY k.name) as key_row_num,
          COUNT(*) OVER (PARTITION BY kr.role_id) as total_keys
        FROM keys_roles kr
        JOIN \`keys\` k ON kr.key_id = k.id
        WHERE kr.workspace_id = ${workspaceId}
      ) key_data ON r.id = key_data.role_id AND key_data.key_row_num <= ${MAX_ITEMS_TO_SHOW}
      LEFT JOIN (
        SELECT 
          rp.role_id,
          p.name as permission_name,
          ROW_NUMBER() OVER (PARTITION BY rp.role_id ORDER BY p.name) as perm_row_num,
          COUNT(*) OVER (PARTITION BY rp.role_id) as total_permissions
        FROM roles_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.workspace_id = ${workspaceId}
      ) perm_data ON r.id = perm_data.role_id AND perm_data.perm_row_num <= ${MAX_ITEMS_TO_SHOW}
      GROUP BY r.id, r.name, r.human_readable, r.description, r.updated_at_m
      ORDER BY r.updated_at_m DESC
    `);

    const rows = result.rows as {
      id: string;
      name: string;
      human_readable: string | null;
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
    const hasMore = rows.length > limit;
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
        slug: row.name,
        name: row.human_readable || "",
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
