import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { toast } from "@unkey/ui";
import { useEffect } from "react";
import { UpsertRoleDialog } from "../../../../upsert-role";
import { useFetchConnectedKeysAndPermsData } from "./hooks/use-fetch-connected-keys-and-perms";

export const EditRole = ({
  role,
  isOpen,
  onClose,
}: {
  role: RoleBasic;
  isOpen: boolean;
  onClose: () => void;
}) => {
  const { permissions, keys, error } = useFetchConnectedKeysAndPermsData(role.roleId);

  useEffect(() => {
    if (error) {
      if (error.data?.code === "NOT_FOUND") {
        toast.error("Role Not Found", {
          description: "The requested role doesn't exist or you don't have access to it.",
        });
      } else if (error.data?.code === "FORBIDDEN") {
        toast.error("Access Denied", {
          description:
            "You don't have permission to view this role. Please contact your administrator.",
        });
      } else if (error.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We were unable to load role details. Please try again or contact support@unkey.dev",
        });
      } else {
        toast.error("Failed to Load Role Details", {
          description: error.message || "An unexpected error occurred. Please try again.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
          },
        });
      }
    }
  }, [error]);
  return (
    <UpsertRoleDialog
      existingRole={{
        id: role.roleId,
        keyIds: keys.map((key) => key.id),
        permissionIds: permissions.map((permission) => permission.id),
        name: role.name,
        description: role.description,
        assignedKeysDetails: keys ?? [],
        assignedPermsDetails: permissions ?? [],
      }}
      isOpen={isOpen}
      onClose={onClose}
    />
  );
};
