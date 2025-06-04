import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";

export const useDeleteRole = (onSuccess: (data: { roleId: string; message: string }) => void) => {
  const trpcUtils = trpc.useUtils();

  const deleteRole = trpc.authorization.roles.delete.useMutation({
    onSuccess(_, variables) {
      trpcUtils.authorization.roles.invalidate();

      toast.success("Role Deleted", {
        description: "The role has been successfully removed from your workspace.",
      });

      onSuccess({
        roleId: variables.roleId,
        message: "Role deleted successfully",
      });
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Role Not Found", {
          description:
            "The role you're trying to delete no longer exists or you don't have access to it.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while deleting your role. Please try again later or contact support.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
          },
        });
      } else {
        toast.error("Failed to Delete Role", {
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
