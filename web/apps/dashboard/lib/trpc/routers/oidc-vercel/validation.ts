import { parseUrnPermissionParts, unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";

const vercelSlugSchema = z
  .string()
  .trim()
  .min(1)
  .max(100)
  .regex(/^[A-Za-z0-9_-]+$/, "Use only letters, numbers, hyphens, and underscores.");

export const oidcVercelInputSchema = z.object({
  name: z.string().trim().min(1).max(256),
  vercelTeamSlug: vercelSlugSchema,
  vercelProjectName: vercelSlugSchema,
  vercelEnvironment: vercelSlugSchema,
  permissions: z
    .array(
      z
        .string()
        .max(512)
        .refine(
          (permission) =>
            permission.startsWith("unkey:")
              ? parseUrnPermissionParts(permission) !== null
              : unkeyPermissionValidation.safeParse(permission).success,
          "Invalid permission.",
        ),
    )
    .max(100)
    .refine(
      (permissions) => new Set(permissions).size === permissions.length,
      "Permissions must be unique.",
    ),
});

export type OIDCVercelInput = z.infer<typeof oidcVercelInputSchema>;
