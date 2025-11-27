import { useTRPC } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

export const useDeletePermission = (
  onSuccess: (data: {
    permissionIds: string[] | string;
    message: string;
  }) => void,
) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const deletePermission = useMutation(trpc.authorization.permissions.delete.mutationOptions({
    onSuccess(data, variables) {
      queryClient.invalidateQueries(trpc.authorization.permissions.pathFilter());
      const permissionCount = data.deletedCount;
      const isPlural = permissionCount > 1;
      toast.success(isPlural ? "Permissions Deleted" : "Permission Deleted", {
        description: isPlural
          ? `${permissionCount} permissions have been successfully removed from your workspace.`
          : "The permission has been successfully removed from your workspace.",
      });
      onSuccess({
        permissionIds: variables.permissionIds,
        message: isPlural
          ? `${permissionCount} permissions deleted successfully`
          : "Permission deleted successfully",
      });
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Permission(s) Not Found", {
          description:
            "One or more permissions you're trying to delete no longer exist or you don't have access to them.",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid Request", {
          description: err.message || "Please provide at least one permission to delete.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while deleting your permissions. Please try again later or contact support.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
          },
        });
      } else {
        toast.error("Failed to Delete Permission(s)", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
          },
        });
      }
    },
  }));
  return deletePermission;
};
