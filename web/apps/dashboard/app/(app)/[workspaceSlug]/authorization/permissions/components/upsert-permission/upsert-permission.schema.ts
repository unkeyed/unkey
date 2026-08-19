import { z } from "zod";

export const permissionNameSchema = z
  .string()
  .min(1, { message: "Permission name must be at least 1 character long" })
  .max(512, { message: "Permission name cannot exceed 512 characters" })
  .refine((name) => !name.match(/^\s|\s$/), {
    error: "Permission name cannot start or end with whitespace",
  })
  .refine((name) => !name.match(/\s{2,}/), {
    error: "Permission name cannot contain consecutive spaces",
  });

// Mirrors the shared permission slug rule in the OpenAPI spec, so a slug
// created here can always be assigned to a key. Allows dotted slugs
// (documents.read), wildcards (documents.*), colons (api:read) and leading
// digits (2fa.read); rejects whitespace. 128 matches permissions.slug.
export const permissionSlugPattern = /^[a-zA-Z0-9_:\-.*]+$/;

export const permissionSlugSchema = z
  .string()
  .trim()
  .min(2, { message: "Permission slug must be at least 2 characters long" })
  .max(128, { message: "Permission slug cannot exceed 128 characters" })
  .refine((slug) => permissionSlugPattern.test(slug), {
    error: "Permission slug can only contain letters, numbers, and the characters _ : - . *",
  });

export const permissionDescriptionSchema = z
  .string()
  .trim()
  .max(512, { message: "Permission description cannot exceed 512 characters" })
  .optional();

export const permissionSchema = z.strictObject({
  permissionId: z.string().startsWith("perm_").optional(),
  name: permissionNameSchema,
  slug: permissionSlugSchema,
  description: permissionDescriptionSchema,
});

export type PermissionFormValues = z.infer<typeof permissionSchema>;
