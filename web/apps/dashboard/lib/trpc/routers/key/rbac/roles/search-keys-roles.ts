import { db, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import {
  LIMIT,
  RolesSearchResponse,
  rolesSearchPayload,
  transformRole,
} from "./schema-with-helpers";

export const searchKeysRoles = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(rolesSearchPayload)
  .output(RolesSearchResponse)
  .query(async ({ ctx, input }) => {
    const { query } = input;
    const workspaceId = ctx.workspace.id;

    if (!query.trim()) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Search query cannot be empty",
      });
    }

    try {
      const searchTerm = `%${query}%`;

      const rolesQuery = await db.query.roles.findMany({
        where: (role, { and, eq, or }) => {
          return and(
            eq(role.workspaceId, workspaceId),
            or(
              sql`${role.id} COLLATE utf8mb4_0900_ai_ci LIKE ${searchTerm}`,
              sql`${role.name} COLLATE utf8mb4_0900_ai_ci LIKE ${searchTerm}`,
              sql`${role.description} COLLATE utf8mb4_0900_ai_ci LIKE ${searchTerm}`,
            ),
          );
        },
        limit: LIMIT,
        orderBy: (roles, { asc }) => [
          asc(roles.name), // Name matches first
          asc(roles.id), // Then by ID for consistency
        ],
        with: {
          keys: {
            with: {
              key: {
                columns: {
                  id: true,
                  name: true,
                },
              },
            },
          },
          permissions: {
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
        columns: {
          id: true,
          name: true,
          description: true,
        },
      });

      return {
        roles: rolesQuery.map(transformRole),
      };
    } catch (error) {
      console.error("Error searching roles:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to search roles. If this issue persists, please contact support.",
      });
    }
  });
