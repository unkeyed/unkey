import { useInvalidateRbacQueries } from "@/hooks/use-invalidate-rbac-queries";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import type { PermissionFormValues } from "../upsert-permission.schema";

export const useCreatePermission = (onSuccess: () => void) => {
  const invalidateRbacQueries = useInvalidateRbacQueries();
  return useMutation({
    mutationFn: async ({ name, slug, description }: PermissionFormValues) => {
      await getUnkeyClient().permissions.createPermission({ name, slug, description });
    },
    onSuccess() {
      invalidateRbacQueries();
      toast.success("Permission Created", { description: "Permission created successfully" });
      onSuccess();
    },
    onError(err) {
      const { message, description } = getErrorToast(err, "Failed to Save Permission");
      toast.error(message, { description });
    },
  });
};
