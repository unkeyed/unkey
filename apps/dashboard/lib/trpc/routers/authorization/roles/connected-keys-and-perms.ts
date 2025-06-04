import { db, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const roleDetailsInput = z.object({
  roleId: z.string().min(1, "Role ID is required"),
});

const roleKey = z.object({
  id: z.string(),
  name: z.string().nullable(),
});
export type RoleKey = z.infer<typeof roleKey>;

const rolePermission = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  description: z.string().nullable(),
});
export type RolePermission = z.infer<typeof rolePermission>;

const roleDetailsResponse = z.object({
  roleId: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  lastUpdated: z.number(),
  keys: z.array(roleKey),
  permissions: z.array(rolePermission),
});

export type RoleDetails = z.infer<typeof roleDetailsResponse>;

export const getConnectedKeysAndPerms = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(roleDetailsInput)
  .output(roleDetailsResponse)
  .query(async ({ ctx, input }) => {
    const { roleId } = input;
    const workspaceId = ctx.workspace.id;

    try {
      // First, verify the role exists in this workspace - security check
      const roleCheck = await db.execute(sql`
        SELECT id, name, description, updated_at_m
        FROM roles 
        WHERE id = ${roleId} AND workspace_id = ${workspaceId}
        LIMIT 1
      `);

      const roleRows = roleCheck.rows as {
        id: string;
        name: string;
        description: string | null;
        updated_at_m: number;
      }[];

      if (roleRows.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Role not found or access denied",
        });
      }

      const role = roleRows[0];

      // Validate required fields
      if (!role.id || !role.name) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Invalid role data retrieved",
        });
      }

      // Fetch all keys for this role
      const keysResult = await db.execute(sql`
        SELECT DISTINCT k.id, k.name
        FROM keys_roles kr
        JOIN \`keys\` k ON kr.key_id = k.id
        WHERE kr.role_id = ${roleId} 
          AND kr.workspace_id = ${workspaceId}
        ORDER BY COALESCE(k.name, k.id)
      `);

      const keyRows = keysResult.rows as {
        id: string;
        name: string | null;
      }[];

      // Fetch all permissions for this role
      const permissionsResult = await db.execute(sql`
        SELECT DISTINCT p.name, p.slug, p.description, p.id
        FROM roles_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = ${roleId} 
          AND rp.workspace_id = ${workspaceId}
          AND p.name IS NOT NULL
        ORDER BY p.name
      `);

      const permissionRows = permissionsResult.rows as {
        id: string;
        name: string;
        slug: string;
        description: string | null;
      }[];

      return {
        roleId: role.id,
        name: role.name,
        description: role.description,
        lastUpdated: Number(role.updated_at_m),
        keys: keyRows.map((row) => {
          if (!row.id) {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message: "Invalid key data retrieved",
            });
          }
          return {
            id: row.id,
            name: row.name,
          };
        }),
        permissions: permissionRows.map((row) => {
          if (!row.name || !row.slug) {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message: "Invalid permission data retrieved",
            });
          }
          return {
            id: row.id,
            name: row.name,
            slug: row.slug,
            description: row.description,
          };
        }),
      };
    } catch (error) {
      // Re-throw TRPCErrors as-is
      if (error instanceof TRPCError) {
        throw error;
      }

      // Handle database connection errors
      if (error instanceof Error && error.message.includes("connection")) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Database connection failed",
        });
      }

      // Handle all other errors
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch role details",
        cause: error,
      });
    }
  });
