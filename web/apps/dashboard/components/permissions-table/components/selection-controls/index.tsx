import { TableDeleteSelectionControls } from "@/components/table-delete-selection-controls";
import { useDeletePermission } from "../actions/components/hooks/use-delete-permission";

type SelectionControlsProps = {
  selectedPermissions: Set<string>;
  setSelectedPermissions: (keys: Set<string>) => void;
};

export const SelectionControls = ({
  selectedPermissions,
  setSelectedPermissions,
}: SelectionControlsProps) => {
  const deletePermission = useDeletePermission(() => {
    setSelectedPermissions(new Set());
  });

  return (
    <TableDeleteSelectionControls
      selectedCount={selectedPermissions.size}
      onClearSelection={() => setSelectedPermissions(new Set())}
      onConfirmDelete={() =>
        deletePermission.mutate({
          permissionIds: Array.from(selectedPermissions),
        })
      }
      isDeleting={deletePermission.isLoading}
      singular="permission"
      plural="permissions"
    />
  );
};
