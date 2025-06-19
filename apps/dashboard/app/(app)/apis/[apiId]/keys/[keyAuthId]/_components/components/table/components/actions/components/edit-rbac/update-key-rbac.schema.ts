import type { KeyPermission } from "@/lib/trpc/routers/key/rbac/connected-roles-and-perms";
import { z } from "zod";

export const updateKeyRbacSchema = z.object({
  keyId: z.string().min(1, "Key ID is required"),
  roleIds: z.array(z.string()),
  directPermissionIds: z.array(z.string()),
});

export type FormValues = z.infer<typeof updateKeyRbacSchema>;

export interface DisplayPermission extends KeyPermission {
  isInherited: boolean;
}
