import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";

export const useUpdateKeyRbac = (
  onSuccess: (data: {
    keyId: string;
    success: boolean;
    rolesAssigned: number;
    permissionsAssigned: number;
  }) => void,
) => {
  const trpcUtils = trpc.useUtils();

  const updateKeyRbac = trpc.key.update.rbac.update.useMutation({
    onSuccess(data) {
      trpcUtils.key.logs.query.invalidate();
      trpcUtils.key.connectedRolesAndPerms.invalidate();

      // Show success toast
      toast.success("Key Permissions and Roles Updated", {
        description: `Updated with ${data.rolesAssigned} roles and ${data.permissionsAssigned} permissions`,
      });

      onSuccess(data);
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Key Not Found", {
          description:
            "The key you're trying to update no longer exists or you don't have access to it.",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid RBAC Configuration", {
          description: `Please check your role and permission selections. ${err.message || ""}`,
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while updating key permissions. Please try again later or contact support.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
          },
        });
      } else {
        toast.error("Failed to Update Key Permissions", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
          },
        });
      }
    },
  });

  return updateKeyRbac;
};
