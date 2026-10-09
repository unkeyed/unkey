import { useInvalidateRbacQueries } from "@/hooks/use-invalidate-rbac-queries";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import type { PermissionFormValues } from "../upsert-permission.schema";

type UpdatePermissionInput = PermissionFormValues & { permissionId: string };

export const useUpdatePermission = (onSuccess: () => void) => {
  const invalidateRbacQueries = useInvalidateRbacQueries();
  return useMutation({
    mutationFn: async ({ permissionId, name, slug, description }: UpdatePermissionInput) => {
      await getUnkeyClient().permissions.updatePermission({
        permission: permissionId,
        name,
        slug,
        description,
      });
    },
    onSuccess() {
      invalidateRbacQueries();
      toast.success("Permission Updated", { description: "Permission updated successfully" });
      onSuccess();
    },
    onError(err) {
      const { message, description } = getErrorToast(err, "Failed to Save Permission");
      toast.error(message, { description });
    },
  });
};
