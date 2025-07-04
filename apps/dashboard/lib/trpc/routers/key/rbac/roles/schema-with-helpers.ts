import type { Role } from "@unkey/db";
import { z } from "zod";

export const LIMIT = 50;

export const rolesQueryPayload = z.object({
  cursor: z.string().optional(),
  limit: z.number().default(LIMIT),
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
  query: z.string().trim().min(1, "Search query cannot be empty"),
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

export const transformRole = (
  role: RoleWithKeysAndPermissions | Pick<Role, "id" | "name" | "description">,
) => ({
  id: role.id,
  name: role.name,
  description: role.description,
  keys:
    "keys" in role
      ? role.keys
          .filter(
            (
              roleKey,
            ): roleKey is typeof roleKey & {
              key: { id: string; name: string | null };
            } => roleKey.key?.id != null,
          )
          .map((roleKey) => ({
            id: roleKey.key.id,
            name: roleKey.key.name,
          }))
      : [],
  permissions:
    "permissions" in role
      ? role.permissions
          .filter(
            (
              rolePermission,
            ): rolePermission is typeof rolePermission & {
              permission: { id: string; name: string };
            } => rolePermission.permission?.id != null,
          )
          .map((rolePermission) => ({
            id: rolePermission.permission.id,
            name: rolePermission.permission.name,
          }))
      : [],
});
