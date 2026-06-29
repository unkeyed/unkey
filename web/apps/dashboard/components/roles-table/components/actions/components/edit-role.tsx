import { UpsertRoleDialog } from "@/app/(app)/[workspaceSlug]/authorization/roles/components/upsert-role";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { toast } from "@unkey/ui";
import { useEffect } from "react";
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
  const { permissions, keys, error, hasData } = useFetchConnectedKeysAndPermsData(role.roleId);

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
            "We were unable to load role details. Please try again or contact support@unkey.com",
        });
      } else {
        toast.error("Failed to Load Role Details", {
          description: error.message || "An unexpected error occurred. Please try again.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    }
  }, [error]);

  // Don't mount the form until we actually have the connected keys/permissions.
  // Gating on data presence (rather than a loading flag) covers every state in
  // which the baseline would be empty: in-flight fetches, fetch errors, and
  // disabled queries for over-limit roles whose cache isn't populated yet. If we
  // rendered in any of those, the dialog would initialize its baseline with empty
  // keyIds/permissionIds and submitting would wipe the role's existing
  // associations. Errors surface via the toast above.
  if (!hasData) {
    return null;
  }

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
