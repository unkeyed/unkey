import { rootKeysQueryPayload } from "@/components/root-keys-table/schema/query-logs.schema";
import { and, asc, count, db, desc, eq, or, schema, sql } from "@/lib/db";
import {
  ratelimit,
  requireWorkspaceAdmin,
  withRatelimit,
  workspaceProcedure,
} from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { parseUrnPermissionParts } from "@unkey/rbac";
import { z } from "zod";
import { rootKeyBaseConditions, rootKeyPermissions } from "./shared";

const PermissionResponse = z.object({
  id: z.string(),
  name: z.string(),
});

const RootKeyResponse = z.object({
  id: z.string(),
  prefix: z.string(),
  start: z.string(),
  end: z.string(),
  createdAt: z.number(),
  lastUsedAt: z.number(),
  lastUpdatedAt: z.number().nullable(),
  expires: z.number().nullable(),
  name: z.string().nullable(),
  permissionSummary: z.object({
    total: z.number(),
    categories: z.record(z.string(), z.number()),
    hasCriticalPerm: z.boolean(),
  }),
  permissions: z.array(PermissionResponse),
});

const RootKeysResponse = z.object({
  keys: z.array(RootKeyResponse),
  hasMore: z.boolean(),
  total: z.number(),
});

type PermissionResponse = z.infer<typeof PermissionResponse>;
type RootKeysResponse = z.infer<typeof RootKeysResponse>;
export type RootKey = z.infer<typeof RootKeyResponse>;

export const LIMIT = 50;
export const MAX_LIMIT = 200;

export const queryRootKeys = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .use(withRatelimit(ratelimit.read))
  .input(rootKeysQueryPayload)
  .output(RootKeysResponse)
  .query(async ({ ctx, input }) => {
    // Build base conditions (used for both count and fetch)
    const baseConditions = rootKeyBaseConditions(ctx.workspace.id);

    // Build filter conditions
    const filterConditions = [];

    // Name filter
    if (input.name && input.name.length > 0) {
      const nameConditions = input.name.map((filter) => {
        if (filter.operator === "is") {
          return eq(schema.keys.name, filter.value);
        }
        if (filter.operator === "contains") {
          return sql`LOWER(${schema.keys.name}) LIKE LOWER(${`%${filter.value}%`})`;
        }
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `Unsupported name operator: ${filter.operator}`,
        });
      });

      if (nameConditions.length === 1) {
        filterConditions.push(nameConditions[0]);
      } else {
        filterConditions.push(or(...nameConditions));
      }
    }

    // Count conditions: base + filters only (total must reflect all matching keys)
    const countConditions =
      filterConditions.length > 0 ? [...baseConditions, ...filterConditions] : baseConditions;

    // Fetch conditions: base + filters
    const fetchConditions =
      filterConditions.length > 0 ? [...baseConditions, ...filterConditions] : baseConditions;

    // Build ORDER BY based on sort input
    const SORT_COLUMN_MAP = {
      name: schema.keys.name,
      createdAt: schema.keys.createdAtM,
      lastUpdatedAt: schema.keys.updatedAtM,
    } as const;
    const sortColumn = SORT_COLUMN_MAP[input.sortBy ?? "createdAt"];
    const sortFn = input.sortOrder === "asc" ? asc : desc;

    const page = input.page ?? 1;
    const pageSize = Math.min(input.limit ?? LIMIT, MAX_LIMIT);

    try {
      const [totalResult, keysResult] = await Promise.all([
        db
          .select({ count: count() })
          .from(schema.keys)
          .where(and(...countConditions)),
        db.query.keys.findMany({
          where: and(...fetchConditions),
          orderBy: [sortFn(sortColumn), sortFn(schema.keys.id)],
          limit: pageSize,
          offset: (page - 1) * pageSize,
          columns: {
            id: true,
            prefix: true,
            start: true,
            end: true,
            createdAtM: true,
            lastUsedAt: true,
            updatedAtM: true,
            expires: true,
            name: true,
          },
          with: {
            permissions: {
              columns: {
                permissionId: true,
              },
              with: {
                permission: {
                  columns: {
                    id: true,
                    name: true,
                  },
                },
              },
            },
          },
        }),
      ]);

      // Transform the data to flatten permissions and add summary
      const keys = keysResult.map((key) => {
        const permissions = rootKeyPermissions(key.permissions);

        const permissionSummary = categorizePermissions(permissions);

        return {
          id: key.id,
          prefix: key.prefix,
          start: key.start,
          end: key.end,
          createdAt: key.createdAtM,
          lastUsedAt: key.lastUsedAt,
          lastUpdatedAt: key.updatedAtM,
          expires: key.expires ? key.expires.getTime() : null,
          name: key.name,
          permissionSummary,
          permissions,
        };
      });

      const totalCount = totalResult[0]?.count ?? 0;
      const response: RootKeysResponse = {
        keys,
        hasMore: page * pageSize < totalCount,
        total: totalCount,
      };

      return response;
    } catch (error) {
      console.error("Error querying root keys:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve Root Keys due to an error. If this issue persists, please contact support@unkey.com with the time this occurred.",
      });
    }
  });

const CRITICAL_PERMISSION_ACTIONS = new Set(["delete", "decrypt"]);

function categorizePermissions(permissions: PermissionResponse[]) {
  if (!Array.isArray(permissions)) {
    throw new Error("Invalid permissions array");
  }

  const categories: Record<string, number> = {};
  let hasCriticalPerm = false;

  for (const permission of permissions) {
    if (!permission?.name || typeof permission.name !== "string") {
      console.warn("Invalid permission object:", permission);
      continue;
    }

    const parsed = parseUrnPermissionParts(permission.name);
    if (parsed === null) {
      console.warn(`Invalid permission format: ${permission.name}`);
      continue;
    }

    const segments = parsed.resourcePath.split("/");
    let category: string;

    if (segments.includes("keys")) {
      category = "Keys";
    } else if (segments.includes("keyspaces")) {
      category = "API";
    } else if (segments.includes("ratelimits")) {
      category = "Ratelimit";
    } else if (segments.includes("rbac")) {
      category = "Permissions";
    } else if (segments.includes("identities")) {
      category = "Identities";
    } else if (segments.includes("projects")) {
      category = "Projects";
    } else {
      category = "Other";
    }

    categories[category] = (categories[category] || 0) + 1;

    if (CRITICAL_PERMISSION_ACTIONS.has(parsed.action)) {
      hasCriticalPerm = true;
    }
  }

  return {
    total: permissions.length,
    categories,
    hasCriticalPerm,
  };
}
