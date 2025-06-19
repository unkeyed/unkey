import { z } from "zod";

export const LIMIT = 50;

const RoleSchema = z.object({
  id: z.string(),
  name: z.string(),
});

const PermissionSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  slug: z.string(),
  roles: z.array(RoleSchema),
});

export const permissionsSearchPayload = z.object({
  query: z.string().trim().min(1, "Search query cannot be empty"),
});

export const permissionsQueryPayload = z.object({
  cursor: z.string().optional(),
  limit: z.number().default(LIMIT),
});

export const PermissionsSearchResponse = z.object({
  permissions: z.array(PermissionSchema),
});

export const PermissionsQueryResponse = z.object({
  permissions: z.array(PermissionSchema),
  hasMore: z.boolean(),
  nextCursor: z.string().nullish(),
});

type PermissionWithRoles = {
  id: string;
  name: string;
  description: string | null;
  slug: string;
  roles: {
    role: { id: string; name: string };
  }[];
};

export const transformPermission = (permission: PermissionWithRoles) => ({
  id: permission.id,
  name: permission.name,
  description: permission.description,
  slug: permission.slug,
  roles: permission.roles
    .filter((rolePermission) => Boolean(rolePermission.role))
    .map((rolePermission) => ({
      id: rolePermission.role.id,
      name: rolePermission.role.name,
    })),
});
