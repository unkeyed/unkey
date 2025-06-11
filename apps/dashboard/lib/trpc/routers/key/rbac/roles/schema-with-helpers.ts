import { z } from "zod";

export const LIMIT = 50;

export const rolesQueryPayload = z.object({
  cursor: z.string().optional(),
});

export const KeySchema = z.object({
  id: z.string(),
  name: z.string().nullable(),
});

export const PermissionSchema = z.object({
  id: z.string(),
  name: z.string(),
});

export const RoleResponseSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  keys: z.array(KeySchema),
  permissions: z.array(PermissionSchema),
});

export const RolesResponse = z.object({
  roles: z.array(RoleResponseSchema),
  hasMore: z.boolean(),
  nextCursor: z.string().nullish(),
});

export const rolesSearchPayload = z.object({
  query: z.string().min(1, "Search query cannot be empty"),
});

export const RolesSearchResponse = z.object({
  roles: z.array(RoleResponseSchema),
});

type RoleWithKeysAndPermissions = {
  id: string;
  name: string;
  description: string | null;
  keys: {
    key: { id: string; name: string | null } | null;
  }[];
  permissions: {
    permission: { id: string; name: string } | null;
  }[];
};

export const transformRole = (role: RoleWithKeysAndPermissions) => ({
  id: role.id,
  name: role.name,
  description: role.description,
  keys: role.keys
    .filter((roleKey) => roleKey.key !== null)
    .map((roleKey) => ({
      id: roleKey.key!.id,
      name: roleKey.key!.name,
    })),
  permissions: role.permissions
    .filter((rolePermission) => rolePermission.permission !== null)
    .map((rolePermission) => ({
      id: rolePermission.permission!.id,
      name: rolePermission.permission!.name,
    })),
});
