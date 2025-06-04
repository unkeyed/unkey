import { z } from "zod";

export const roleNameSchema = z
  .string()
  .trim()
  .min(2, { message: "Role name must be at least 2 characters long" })
  .max(60, { message: "Role name cannot exceed 64 characters" })
  .refine((name) => !name.match(/^\s|\s$/), {
    message: "Role name cannot start or end with whitespace",
  })
  .refine((name) => !name.match(/\s{2,}/), {
    message: "Role name cannot contain consecutive spaces",
  });

export const roleDescriptionSchema = z
  .string()
  .trim()
  .max(30, { message: "Role description cannot exceed 30 characters" })
  .optional();

export const keyIdsSchema = z
  .array(
    z.string().refine((id) => id.startsWith("key_"), {
      message: "Each key ID must start with 'key_'",
    }),
  )
  .default([])
  .transform((ids) => [...new Set(ids)]) // Remove duplicates
  .optional();

export const permissionIdsSchema = z
  .array(
    z.string().refine((id) => id.startsWith("perm_"), {
      message: "Each permission ID must start with 'perm_'",
    }),
  )
  .min(1, { message: "Role must have at least one permission assigned" })
  .transform((ids) => [...new Set(ids)]) // Remove duplicates
  .optional();

export const rbacRoleSchema = z
  .object({
    roleId: z.string().startsWith("role_").optional(), // If provided, it's an update
    roleName: roleNameSchema,
    roleDescription: roleDescriptionSchema,
    keyIds: keyIdsSchema,
    permissionIds: permissionIdsSchema,
  })
  .strict({ message: "Unknown fields are not allowed in role definition" });

export type FormValues = z.infer<typeof rbacRoleSchema>;
