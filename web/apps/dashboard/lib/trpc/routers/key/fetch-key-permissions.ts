import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

// Schema for keyAuth object
const KeyAuthSchema = z.object({
  pk: z.number(),
  id: z.string(),
  workspaceId: z.string(),
  createdAtM: z.number(),
  updatedAtM: z.number().nullable(),
  deletedAtM: z.number().nullable(),
  storeEncryptedKeys: z.boolean(),
  defaultPrefix: z.string().nullable(),
  defaultBytes: z.number().nullable(),
  sizeApprox: z.number(),
  sizeLastUpdatedAt: z.number(),
});

// Schema for permission object
const PermissionSchema = z.object({
  pk: z.number(),
  keyId: z.string(),
  permissionId: z.string(),
  workspaceId: z.string(),
  createdAtM: z.number(),
  updatedAtM: z.number().nullable(),
});

// Schema for role with nested permissions
const RoleWithPermissionsSchema = z.object({
  pk: z.number(),
  keyId: z.string(),
  roleId: z.string(),
  workspaceId: z.string(),
  createdAtM: z.number(),
  updatedAtM: z.number().nullable(),
  role: z.object({
    pk: z.number(),
    id: z.string(),
    workspaceId: z.string(),
    name: z.string(),
    description: z.string().nullable(),
    createdAtM: z.number(),
    updatedAtM: z.number().nullable(),
    permissions: z.array(
      z.object({
        pk: z.number(),
        roleId: z.string(),
        permissionId: z.string(),
        workspaceId: z.string(),
        createdAtM: z.number(),
        updatedAtM: z.number().nullable(),
        permission: z.object({
          pk: z.number(),
          id: z.string(),
          workspaceId: z.string(),
          name: z.string(),
          slug: z.string(),
          description: z.string().nullable(),
          createdAtM: z.number(),
          updatedAtM: z.number().nullable(),
        }),
      }),
    ),
  }),
});

// Schema for workspace roles
const WorkspaceRoleSchema = z.object({
  pk: z.number(),
  id: z.string(),
  workspaceId: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  createdAtM: z.number(),
  updatedAtM: z.number().nullable(),
  permissions: z.array(
    z.object({
      pk: z.number(),
      roleId: z.string(),
      permissionId: z.string(),
      workspaceId: z.string(),
      createdAtM: z.number(),
      updatedAtM: z.number().nullable(),
    }),
  ),
});

// Schema for workspace permissions
const WorkspacePermissionSchema = z.object({
  pk: z.number(),
  id: z.string(),
  workspaceId: z.string(),
  name: z.string(),
  slug: z.string(),
  description: z.string().nullable(),
  createdAtM: z.number(),
  updatedAtM: z.number().nullable(),
  roles: z.array(
    z.object({
      pk: z.number(),
      roleId: z.string(),
      permissionId: z.string(),
      workspaceId: z.string(),
      createdAtM: z.number(),
      updatedAtM: z.number().nullable(),
    }),
  ),
});

const KeyPermissionsResponseSchema = z.object({
  keyId: z.string(),
  keyAuth: KeyAuthSchema,
  roles: z.array(RoleWithPermissionsSchema),
  directPermissions: z.array(PermissionSchema),
  workspace: z.object({
    roles: z.array(WorkspaceRoleSchema),
    permissions: z.object({
      roles: z.array(WorkspacePermissionSchema),
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
        columns: { id: true },
      });

      if (!keyAuth) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Keyspace not found or not authorized",
        });
      }

      const key = await db.query.keys.findFirst({
        where: and(
          eq(schema.keys.id, input.keyId),
          eq(schema.keys.workspaceId, ctx.workspace.id),
          isNull(schema.keys.deletedAtM),
        ),
        columns: {
          id: true,
          remaining: true,
        },
        with: {
          keyAuth: {
            columns: {
              pk: true,
              id: true,
              workspaceId: true,
              projectId: true,
              createdAtM: true,
              updatedAtM: true,
              deletedAtM: true,
              storeEncryptedKeys: true,
              defaultPrefix: true,
              defaultBytes: true,
              sizeApprox: true,
              sizeLastUpdatedAt: true,
            },
          },
          roles: {
            columns: {
              pk: true,
              keyId: true,
              roleId: true,
              workspaceId: true,
              createdAtM: true,
              updatedAtM: true,
            },
            with: {
              role: {
                columns: {
                  pk: true,
                  id: true,
                  workspaceId: true,
                  projectId: true,
                  name: true,
                  description: true,
                  createdAtM: true,
                  updatedAtM: true,
                },
                with: {
                  permissions: {
                    columns: {
                      pk: true,
                      roleId: true,
                      permissionId: true,
                      workspaceId: true,
                      createdAtM: true,
                      updatedAtM: true,
                    },
                    with: {
                      permission: {
                        columns: {
                          pk: true,
                          id: true,
                          workspaceId: true,
                          projectId: true,
                          name: true,
                          slug: true,
                          description: true,
                          createdAtM: true,
                          updatedAtM: true,
                        },
                      },
                    },
                  },
                },
              },
            },
          },
          permissions: {
            columns: {
              pk: true,
              keyId: true,
              permissionId: true,
              workspaceId: true,
              createdAtM: true,
              updatedAtM: true,
            },
          },
          workspace: {
            columns: { id: true },
            with: {
              roles: {
                columns: {
                  pk: true,
                  id: true,
                  workspaceId: true,
                  projectId: true,
                  name: true,
                  description: true,
                  createdAtM: true,
                  updatedAtM: true,
                },
                with: {
                  permissions: {
                    columns: {
                      pk: true,
                      roleId: true,
                      permissionId: true,
                      workspaceId: true,
                      createdAtM: true,
                      updatedAtM: true,
                    },
                  },
                },
              },
              permissions: {
                columns: {
                  pk: true,
                  id: true,
                  workspaceId: true,
                  projectId: true,
                  name: true,
                  slug: true,
                  description: true,
                  createdAtM: true,
                  updatedAtM: true,
                },
                with: {
                  roles: {
                    columns: {
                      pk: true,
                      roleId: true,
                      permissionId: true,
                      workspaceId: true,
                      createdAtM: true,
                      updatedAtM: true,
                    },
                  },
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
          "Failed to retrieve key permissions. If this issue persists, please contact support@unkey.com with the time this occurred.",
      });
    }
  });
