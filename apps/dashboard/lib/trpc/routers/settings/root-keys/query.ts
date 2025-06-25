import { rootKeysQueryPayload } from "@/app/(app)/settings/root-keys-v2/components/table/query-logs.schema";
import { and, count, db, desc, eq, isNull, lt, schema } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const PermissionResponse = z.object({
  id: z.string(),
  name: z.string(),
});

const RootKeyResponse = z.object({
  id: z.string(),
  start: z.string(),
  createdAt: z.number(),
  lastUpdatedAt: z.number().nullable(),
  name: z.string().nullable(),
  permissionSummary: z.object({
    total: z.number(),
    categories: z.record(z.number()),
    hasCriticalPerm: z.boolean(),
  }),
  permissions: z.array(PermissionResponse),
});

const RootKeysResponse = z.object({
  keys: z.array(RootKeyResponse),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.number().int().optional(),
});

type PermissionResponse = z.infer<typeof PermissionResponse>;
type RootKeysResponse = z.infer<typeof RootKeysResponse>;
export type RootKey = z.infer<typeof RootKeyResponse>;

export const LIMIT = 50;

export const queryRootKeys = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(rootKeysQueryPayload)
  .output(RootKeysResponse)
  .query(async ({ ctx, input }) => {
    // Build base conditions
    const baseConditions = [
      eq(schema.keys.forWorkspaceId, ctx.workspace.id),
      isNull(schema.keys.deletedAtM),
    ];

    // Add cursor condition for pagination
    if (input.cursor && typeof input.cursor === "number") {
      baseConditions.push(lt(schema.keys.createdAtM, input.cursor));
    }

    try {
      const [totalResult, keysResult] = await Promise.all([
        db
          .select({ count: count() })
          .from(schema.keys)
          .where(and(...baseConditions)),
        db.query.keys.findMany({
          where: and(...baseConditions),
          orderBy: [desc(schema.keys.createdAtM)],
          limit: LIMIT + 1, // Get one extra to check if there are more
          columns: {
            id: true,
            start: true,
            createdAtM: true,
            updatedAtM: true,
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

      // Check if we have more results
      const hasMore = keysResult.length > LIMIT;
      const keysWithoutExtra = hasMore ? keysResult.slice(0, LIMIT) : keysResult;

      // Transform the data to flatten permissions and add summary
      const keys = keysWithoutExtra.map((key) => {
        const permissions = key.permissions
          .map((p) => p.permission)
          .filter(Boolean)
          .map((permission) => ({
            id: permission.id,
            name: permission.name,
          }));

        const permissionSummary = categorizePermissions(permissions);

        return {
          id: key.id,
          start: key.start,
          createdAt: key.createdAtM,
          lastUpdatedAt: key.updatedAtM,
          name: key.name,
          permissionSummary,
          permissions,
        };
      });

      const response: RootKeysResponse = {
        keys,
        hasMore,
        total: totalResult[0]?.count ?? 0,
        nextCursor: keys.length > 0 ? keys[keys.length - 1].createdAt : undefined,
      };

      return response;
    } catch (error) {
      console.error("Error querying root keys:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve root keys due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
      });
    }
  });

const CRITICAL_PERMISSION_PATTERNS = ["delete", "decrypt", "remove"] as const;

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

    // Extract category from permission name (e.g., "api.*.create_key" -> "api")
    const parts = permission.name.split(".");
    if (parts.length < 3) {
      console.warn(`Invalid permission format: ${permission.name}`);
      continue;
    }

    const [identifier, , action] = parts;
    let category: string;

    switch (identifier) {
      case "api":
        // Separate API permissions from key permissions
        if (action.includes("key")) {
          category = "Keys";
        } else {
          category = "API";
        }
        break;
      case "ratelimit":
        category = "Ratelimit";
        break;
      case "rbac":
        category = "Permissions";
        break;
      case "identity":
        category = "Identities";
        break;
      default:
        category = "Other";
        console.warn(`Unknown permission identifier: ${identifier}`);
    }

    categories[category] = (categories[category] || 0) + 1;

    // Check for critical permissions
    const permissionName = permission.name.toLowerCase();
    if (CRITICAL_PERMISSION_PATTERNS.some((pattern) => permissionName.includes(pattern))) {
      hasCriticalPerm = true;
    }
  }

  return {
    total: permissions.length,
    categories,
    hasCriticalPerm,
  };
}
