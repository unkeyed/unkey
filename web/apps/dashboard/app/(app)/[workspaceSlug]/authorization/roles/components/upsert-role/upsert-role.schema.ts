import { z } from "zod";

// 128 matches the role name cap in the v2 API and permissions.slug, so a role
// created here can always be assigned to a key. The roles.name column is wider
// (varchar(512)), but names above this cap could not be used at key creation.
export const roleNameSchema = z
  .string()
  .trim()
  .min(1, { message: "Role name must be at least 1 characters long" })
  .max(128, { message: "Role name cannot exceed 128 characters" })
  .refine((name) => !name.match(/^\s|\s$/), {
    error: "Role name cannot start or end with whitespace",
  })
  .refine((name) => !name.match(/\s{2,}/), {
    error: "Role name cannot contain consecutive spaces",
  });

export const roleDescriptionSchema = z
  .string()
  .trim()
  .max(512, { message: "Role description cannot exceed 512 characters" })
  .optional();

export const keyIdsSchema = z.array(z.string()).transform((ids) => [...new Set(ids)]);

export const permissionIdsSchema = z.array(z.string()).transform((ids) => [...new Set(ids)]);

export const rbacRoleSchema = z.strictObject({
  roleId: z.string().startsWith("role_").optional(),
  roleName: roleNameSchema,
  roleDescription: roleDescriptionSchema,
  keyIds: keyIdsSchema.optional(),
  permissionIds: permissionIdsSchema.optional(),
});

export type FormValues = z.infer<typeof rbacRoleSchema>;
