import { z } from "zod";

export const permissionNameSchema = z
  .string()
  .min(1, { message: "Permission name must be at least 1 character long" })
  .max(512, { message: "Permission name cannot exceed 512 characters" })
  .refine((name) => !name.match(/^\s|\s$/), {
    message: "Permission name cannot start or end with whitespace",
  })
  .refine((name) => !name.match(/\s{2,}/), {
    message: "Permission name cannot contain consecutive spaces",
  });

export const permissionSlugSchema = z
  .string()
  .trim()
  .min(2, { message: "Permission slug must be at least 2 characters long" })
  .max(128, { message: "Permission slug cannot exceed 128 characters" });

export const permissionDescriptionSchema = z
  .string()
  .trim()
  .max(512, { message: "Permission description cannot exceed 128 characters" })
  .optional();

export const permissionSchema = z
  .object({
    permissionId: z.string().startsWith("perm_").optional(),
    name: permissionNameSchema,
    slug: permissionSlugSchema,
    description: permissionDescriptionSchema,
  })
  .strict({
    message: "Unknown fields are not allowed in permission definition",
  });

export type PermissionFormValues = z.infer<typeof permissionSchema>;
