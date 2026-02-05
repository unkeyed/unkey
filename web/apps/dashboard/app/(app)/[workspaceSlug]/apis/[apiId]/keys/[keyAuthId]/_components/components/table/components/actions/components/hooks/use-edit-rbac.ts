import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useUpdateKeyRbac = (
  onSuccess: (data: {
    keyId: string;
    success: boolean;
    rolesAssigned: number;
    directPermissionsAssigned: number;
    totalEffectivePermissions: number;
  }) => void,
) => {
  const trpcUtils = trpc.useUtils();

  const updateKeyRbac = trpc.key.update.rbac.update.useMutation({
    onSuccess(data) {
      trpcUtils.key.connectedRolesAndPerms.invalidate();
      trpcUtils.api.keys.list.invalidate();

      const { rolesAssigned, directPermissionsAssigned, totalEffectivePermissions } = data;

      const parts = [];
      if (rolesAssigned > 0) {
        parts.push(`${rolesAssigned} role${rolesAssigned > 1 ? "s" : ""}`);
      }
      if (directPermissionsAssigned > 0) {
        parts.push(
          `${directPermissionsAssigned} direct permission${
            directPermissionsAssigned > 1 ? "s" : ""
          }`,
        );
      }

      const description =
        parts.length > 0
          ? `Updated with ${parts.join(
              " and ",
            )} (${totalEffectivePermissions} total effective permissions)`
          : "All roles and permissions have been removed from this key";

      toast.success("Key RBAC Updated", {
        description,
        duration: 5000,
      });

      onSuccess(data);
    },

    onError(err) {
      console.error("Key RBAC update failed:", err);

      if (err.data?.code === "NOT_FOUND") {
        toast.error("Key Update Failed", {
          description:
            "We are unable to find the correct key. Please try again or contact support@unkey.com.",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        let description = "Please check your selections and try again.";

        if (err.message.includes("roles")) {
          description =
            "One or more selected roles do not exist in this workspace. Please refresh and try again.";
        } else if (err.message.includes("permissions")) {
          description =
            "One or more selected permissions do not exist in this workspace. Please refresh and try again.";
        }

        toast.error("Invalid Selection", {
          description,
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We are unable to update RBAC for this key. Please try again or contact support@unkey.com",
        });
      } else {
        toast.error("Failed to Update Key RBAC", {
          description:
            err.message ||
            "An unexpected error occurred while updating roles and permissions. Please try again or contact support@unkey.com",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });

  return updateKeyRbac;
};
