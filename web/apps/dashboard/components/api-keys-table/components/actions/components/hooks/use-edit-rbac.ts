import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import type { FormValues } from "../edit-rbac/update-key-rbac.schema";

type UpdateKeyRbacResult = {
  keyId: string;
  success: boolean;
  rolesAssigned: number;
  directPermissionsAssigned: number;
  totalEffectivePermissions: number;
};

type UpdateKeyRbacVariables = FormValues & {
  totalEffectivePermissions: number;
};

export const useUpdateKeyRbac = (
  onSuccess: (data: UpdateKeyRbacResult) => void,
) => {
  const trpcUtils = trpc.useUtils();

  const updateKeyRbac = useMutation<UpdateKeyRbacResult, unknown, UpdateKeyRbacVariables>({
    mutationFn: async (variables) => {
      await getUnkeyClient().keys.updateKey({
        keyId: variables.keyId,
        roles: variables.roleNames,
        permissions: variables.directPermissionSlugs,
      });

      return {
        keyId: variables.keyId,
        success: true,
        rolesAssigned: variables.roleNames.length,
        directPermissionsAssigned: variables.directPermissionSlugs.length,
        totalEffectivePermissions: variables.totalEffectivePermissions,
      };
    },
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
      toast.error("Failed to Update Key RBAC", {
        description: getErrorMessage(err),
        action: {
          label: "Contact Support",
          onClick: () => window.open("mailto:support@unkey.com", "_blank"),
        },
      });
    },
  });

  return updateKeyRbac;
};
