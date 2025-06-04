import type { MenuItem } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { toast } from "@/components/ui/toaster";
import type { Roles } from "@/lib/trpc/routers/authorization/roles/query";
import { Clone, PenWriting3, Trash } from "@unkey/icons";
import { DeleteRole } from "./components/delete-role";
import { EditRole } from "./components/edit-role";

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
            toast.success("Role data copied to clipboard");
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
      ActionComponent: (props) => <DeleteRole {...props} roleDetails={role} />,
    },
  ];
};
