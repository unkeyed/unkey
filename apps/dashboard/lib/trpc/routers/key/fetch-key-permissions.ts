import { and, db, eq, isNull, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const KeyPermissionsResponseSchema = z.object({
  keyId: z.string(),
  keyAuth: z.any(),
  roles: z.array(z.any()),
  directPermissions: z.array(z.any()),
  workspace: z.object({
    roles: z.array(z.any()),
    permissions: z.object({
      roles: z.array(z.any()),
    }),
  }),
  remainingCredit: z.number().nullish(),
});

const KeyPermissionsRequestSchema = z.object({
  keyId: z.string(),
  keyspaceId: z.string(),
});

export const fetchKeyPermissions = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(KeyPermissionsRequestSchema)
  .output(KeyPermissionsResponseSchema)
  .query(async ({ ctx, input }) => {
    try {
      const keyAuth = await db.query.keyAuth.findFirst({
        where: (keyAuth, { and, eq }) =>
          and(eq(keyAuth.id, input.keyspaceId), eq(keyAuth.workspaceId, ctx.workspace.id)),
      });

      if (!keyAuth) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Keyspace not found or not authorized",
        });
      }

      const key = await db.query.keys.findFirst({
        where: and(eq(schema.keys.id, input.keyId), isNull(schema.keys.deletedAtM)),
        with: {
          keyAuth: true,
          roles: {
            with: {
              role: {
                with: {
                  permissions: {
                    with: {
                      permission: true,
                    },
                  },
                },
              },
            },
          },
          permissions: true,
          workspace: {
            with: {
              roles: {
                with: {
                  permissions: true,
                },
              },
              permissions: {
                with: {
                  roles: true,
                },
              },
            },
          },
        },
      });

      if (!key) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Key not found",
        });
      }

      return {
        keyId: key.id,
        keyAuth: key.keyAuth,
        roles: key.roles,
        directPermissions: key.permissions,
        workspace: {
          roles: key.workspace.roles,
          permissions: {
            roles: key.workspace.permissions,
          },
        },
        remainingCredit: key.remaining,
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Error retrieving key permissions:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve key permissions. If this issue persists, please contact support@unkey.dev with the time this occurred.",
      });
    }
  });
