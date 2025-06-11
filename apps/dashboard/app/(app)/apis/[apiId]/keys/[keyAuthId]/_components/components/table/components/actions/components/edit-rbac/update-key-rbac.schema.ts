import { z } from "zod";

export const updateKeyRbacSchema = z.object({
  keyId: z.string().min(1, "Key ID is required"),
  roleIds: z.array(z.string()).optional().default([]),
  permissionIds: z.array(z.string()).optional().default([]),
});

export type FormValues = z.infer<typeof updateKeyRbacSchema>;
