import type { MenuItem } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { toast } from "@/components/ui/toaster";
import type { Roles } from "@/lib/trpc/routers/authorization/roles/query";
import { Clone, PenWriting3, Trash } from "@unkey/icons";
import { useEffect } from "react";
import { UpsertRoleDialog } from "../../../upsert-role";
import { DeleteKey } from "./components/delete-key";
import { useFetchConnectedKeysAndPerms } from "./components/hooks/use-fetch-connected-keys-and-perms";

export const getRolesTableActionItems = (role: Roles): MenuItem[] => {
  return [
    {
      id: "edit-role",
      label: "Edit role...",
      icon: <PenWriting3 size="md-regular" />,
      ActionComponent: (props) => <EditRole role={role} {...props} />,
    },
    {
      id: "copy",
      label: "Copy role",
      className: "mt-1",
      icon: <Clone size="md-regular" />,
      onClick: () => {
        navigator.clipboard
          .writeText(JSON.stringify(role))
          .then(() => {
            toast.success("Key ID copied to clipboard");
          })
          .catch((error) => {
            console.error("Failed to copy to clipboard:", error);
            toast.error("Failed to copy to clipboard");
          });
      },
      divider: true,
    },
    {
      id: "delete-role",
      label: "Delete role",
      icon: <Trash size="md-regular" />,
      ActionComponent: (props) => <DeleteKey {...props} keyDetails={role} />,
    },
  ];
};

const EditRole = ({
  role,
  isOpen,
  onClose,
}: {
  role: Roles;
  isOpen: boolean;
  onClose: () => void;
}) => {
  const { permissions, keys, error } = useFetchConnectedKeysAndPerms(role.roleId);

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
