import { useTRPC } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

export const useUpdateKeyRbac = (
  onSuccess: (data: {
    keyId: string;
    success: boolean;
    rolesAssigned: number;
    directPermissionsAssigned: number;
    totalEffectivePermissions: number;
  }) => void,
) => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const updateKeyRbac = useMutation(trpc.key.update.rbac.update.mutationOptions({
    onSuccess(data) {
      queryClient.invalidateQueries(trpc.key.connectedRolesAndPerms.pathFilter());
      queryClient.invalidateQueries(trpc.api.keys.list.pathFilter());

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
            "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
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
            "We are unable to update RBAC for this key. Please try again or contact support@unkey.dev",
        });
      } else {
        toast.error("Failed to Update Key RBAC", {
          description:
            err.message ||
            "An unexpected error occurred while updating roles and permissions. Please try again or contact support@unkey.dev",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
          },
        });
      }
    },
  }));

  return updateKeyRbac;
};
