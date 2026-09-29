import { trpc } from "@/lib/trpc/client";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import * as errors from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";

export const useDeletePermission = (onDone: (remainingPermissionIds: string[]) => void) => {
  const trpcUtils = trpc.useUtils();
  return useMutation({
    mutationFn: async (permissionIds: string[]) => {
      const unkey = getUnkeyClient();
      const results = await Promise.allSettled(
        permissionIds.map((permission) => unkey.permissions.deletePermission({ permission })),
      );
      const failures = results.flatMap<{ permissionId: string; error: unknown }>((result, index) =>
        result.status === "rejected"
          ? [{ permissionId: permissionIds[index], error: result.reason }]
          : [],
      );
      return { deletedCount: permissionIds.length - failures.length, failures };
    },
    onSuccess({ deletedCount, failures }, permissionIds) {
      trpcUtils.authorization.invalidate();
      onDone(
        failures
          .filter((failure) => !(failure.error instanceof errors.NotFoundErrorResponse))
          .map((failure) => failure.permissionId),
      );

      if (failures.length === 0) {
        const isPlural = deletedCount > 1;
        toast.success(isPlural ? "Permissions Deleted" : "Permission Deleted", {
          description: isPlural
            ? `${deletedCount} permissions have been successfully removed from your workspace.`
            : "The permission has been successfully removed from your workspace.",
        });
        return;
      }

      const { message, description } = getErrorToast(
        failures[0].error,
        "Failed to Delete Permission",
      );
      toast.error(
        permissionIds.length === 1
          ? message
          : `Deleted ${deletedCount} of ${permissionIds.length} Permissions`,
        { description },
      );
    },
  });
};
