import { unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";

export const createRootKeyInput = z.object({
  name: z.string().optional(),
  permissions: z.array(unkeyPermissionValidation).min(1, {
    error: "You need to add at least one permissions.",
  }),
});

export const updateRootKeyPermissionsInput = z.object({
  keyId: z.string(),
  permissions: z.array(unkeyPermissionValidation).min(1, {
    error: "You need to add at least one permission.",
  }),
});
