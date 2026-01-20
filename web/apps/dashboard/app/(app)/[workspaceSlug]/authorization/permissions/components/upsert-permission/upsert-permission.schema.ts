import { z } from "zod";

export const permissionNameSchema = z
  .string()
  .min(2, {
    error: "Permission name must be at least 2 characters long",
  })
  .max(60, {
    error: "Permission name cannot exceed 60 characters",
  })
  .refine((name) => !name.match(/^\s|\s$/), {
    error: "Permission name cannot start or end with whitespace",
  })
  .refine((name) => !name.match(/\s{2,}/), {
    error: "Permission name cannot contain consecutive spaces",
  });

export const permissionSlugSchema = z
  .string()
  .trim()
  .min(2, {
    error: "Permission slug must be at least 2 characters long",
  })
  .max(50, {
    error: "Permission slug cannot exceed 50 characters",
  });

export const permissionDescriptionSchema = z
  .string()
  .trim()
  .max(200, {
    error: "Permission description cannot exceed 200 characters",
  })
  .optional();

export const permissionSchema = z.strictObject({
  permissionId: z.string().startsWith("perm_").optional(),
  name: permissionNameSchema,
  slug: permissionSlugSchema,
  description: permissionDescriptionSchema,
});

export type PermissionFormValues = z.infer<typeof permissionSchema>;
