import { trpc } from "@/lib/trpc/client";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import * as errors from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";

export const useDeleteRole = (onDone: (remainingRoleIds: string[]) => void) => {
  const trpcUtils = trpc.useUtils();
  return useMutation({
    mutationFn: async (roleIds: string[]) => {
      const unkey = getUnkeyClient();
      const results = await Promise.allSettled(
        roleIds.map((role) => unkey.permissions.deleteRole({ role })),
      );
      const failures = results.flatMap<{ roleId: string; error: unknown }>((result, index) =>
        result.status === "rejected" ? [{ roleId: roleIds[index], error: result.reason }] : [],
      );
      return { deletedCount: roleIds.length - failures.length, failures };
    },
    onSuccess({ deletedCount, failures }, roleIds) {
      trpcUtils.authorization.invalidate();
      onDone(
        failures
          .filter((failure) => !(failure.error instanceof errors.NotFoundErrorResponse))
          .map((failure) => failure.roleId),
      );

      if (failures.length === 0) {
        const isPlural = deletedCount > 1;
        toast.success(isPlural ? "Roles Deleted" : "Role Deleted", {
          description: isPlural
            ? `${deletedCount} roles have been successfully removed from your workspace.`
            : "The role has been successfully removed from your workspace.",
        });
        return;
      }

      const { message, description } = getErrorToast(failures[0].error, "Failed to Delete Role");
      toast.error(
        roleIds.length === 1 ? message : `Deleted ${deletedCount} of ${roleIds.length} Roles`,
        { description },
      );
    },
  });
};
