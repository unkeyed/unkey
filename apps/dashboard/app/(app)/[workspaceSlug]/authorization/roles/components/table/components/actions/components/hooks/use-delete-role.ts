import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useDeleteRole = (
  onSuccess: (data: { roleIds: string[] | string; message: string }) => void,
) => {
  const trpcUtils = trpc.useUtils();
  const deleteRole = trpc.authorization.roles.delete.useMutation({
    onSuccess(data, variables) {
      trpcUtils.authorization.roles.invalidate();

      const roleCount = data.deletedCount;
      const isPlural = roleCount > 1;

      toast.success(isPlural ? "Roles Deleted" : "Role Deleted", {
        description: isPlural
          ? `${roleCount} roles have been successfully removed from your workspace.`
          : "The role has been successfully removed from your workspace.",
      });

      onSuccess({
        roleIds: variables.roleIds,
        message: isPlural ? `${roleCount} roles deleted successfully` : "Role deleted successfully",
      });
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Role(s) Not Found", {
          description:
            "One or more roles you're trying to delete no longer exist or you don't have access to them.",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid Request", {
          description: err.message || "Please provide at least one role to delete.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while deleting your roles. Please try again later or contact support.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
          },
        });
      } else {
        toast.error("Failed to Delete Role(s)", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
          },
        });
      }
    },
  });

  return deleteRole;
};
