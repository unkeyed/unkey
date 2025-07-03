import type { Permission } from "@unkey/db";
import { z } from "zod";

export const LIMIT = 50;

export const permissionsQueryPayload = z.object({
  cursor: z.string().optional(),
  limit: z.number().default(LIMIT),
});

export const PermissionResponseSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  slug: z.string(),
});

export const PermissionsQueryResponse = z.object({
  permissions: z.array(PermissionResponseSchema),
  hasMore: z.boolean(),
  nextCursor: z.string().nullish(),
});

export const permissionsSearchPayload = z.object({
  query: z.string().min(1, "Search query cannot be empty"),
});

export const PermissionsSearchResponse = z.object({
  permissions: z.array(PermissionResponseSchema),
});

export const transformPermission = (
  permission: Pick<Permission, "id" | "name" | "description" | "slug">,
) => ({
  id: permission.id,
  name: permission.name,
  description: permission.description,
  slug: permission.slug,
});
