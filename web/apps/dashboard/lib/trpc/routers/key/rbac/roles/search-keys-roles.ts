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
        where: {
          workspaceId,
          RAW: (table) =>
            sql`(${table.id} LIKE ${searchTerm} OR ${table.name} LIKE ${searchTerm} OR ${table.description} LIKE ${searchTerm})`,
        },
        limit: LIMIT,
        orderBy: { name: "asc", id: "asc" },
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
