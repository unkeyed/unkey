import { useInvalidateRbacQueries } from "@/hooks/use-invalidate-rbac-queries";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import type { FormValues } from "../upsert-role.schema";

type UpdateRoleInput = FormValues & {
  roleId: string;
  initialKeyIds: string[];
  initialPermissionIds: string[];
};

export const useUpdateRole = (onSuccess: () => void) => {
  const invalidateRbacQueries = useInvalidateRbacQueries();
  return useMutation({
    mutationFn: async ({
      roleId,
      roleName,
      roleDescription,
      keyIds,
      permissionIds,
      initialKeyIds,
      initialPermissionIds,
    }: UpdateRoleInput) => {
      const unkey = getUnkeyClient();
      await unkey.permissions.updateRole({
        role: roleId,
        name: roleName,
        description: roleDescription,
      });

      const permissionSet = new Set(permissionIds ?? initialPermissionIds);
      const permissionsChanged =
        permissionSet.symmetricDifference(new Set(initialPermissionIds)).size > 0;
      if (permissionsChanged) {
        const permissions = await Promise.all(
          Array.from(permissionSet, async (permission) => {
            const { data } = await unkey.permissions.getPermission({ permission });
            return data.slug;
          }),
        );
        await unkey.permissions.setRolePermissions({ role: roleId, permissions });
      }

      const keySet = new Set(keyIds ?? initialKeyIds);
      const initialKeySet = new Set(initialKeyIds);
      const results = await Promise.allSettled([
        ...Array.from(keySet.difference(initialKeySet), (keyId) =>
          unkey.keys.addRoles({ keyId, roles: [roleName] }),
        ),
        ...Array.from(initialKeySet.difference(keySet), (keyId) =>
          unkey.keys.removeRoles({ keyId, roles: [roleName] }),
        ),
      ]);
      const failures = results
        .filter((result) => result.status === "rejected")
        .map((result) => result.reason);
      return { keyCount: results.length, failures };
    },
    onSuccess({ keyCount, failures }) {
      invalidateRbacQueries();
      onSuccess();

      if (failures.length === 0) {
        toast.success("Role Updated", { description: "Role updated successfully" });
        return;
      }

      const { description } = getErrorToast(failures[0], "Failed to Update Keys");
      toast.error(`Role Updated, ${failures.length} of ${keyCount} Keys Not Updated`, {
        description,
      });
    },
    onError(err) {
      const { message, description } = getErrorToast(err, "Failed to Save Role");
      toast.error(message, { description });
    },
  });
};
