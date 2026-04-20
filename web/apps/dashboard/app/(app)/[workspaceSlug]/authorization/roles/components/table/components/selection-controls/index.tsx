import { TableDeleteSelectionControls } from "@/components/table-delete-selection-controls";
import { useDeleteRole } from "../actions/components/hooks/use-delete-role";

type SelectionControlsProps = {
  selectedRoles: Set<string>;
  setSelectedRoles: (keys: Set<string>) => void;
};

export const SelectionControls = ({ selectedRoles, setSelectedRoles }: SelectionControlsProps) => {
  const deleteRole = useDeleteRole(() => {
    setSelectedRoles(new Set());
  });

  return (
    <TableDeleteSelectionControls
      selectedCount={selectedRoles.size}
      onClearSelection={() => setSelectedRoles(new Set())}
      onConfirmDelete={() =>
        deleteRole.mutate({
          roleIds: Array.from(selectedRoles),
        })
      }
      isDeleting={deleteRole.isLoading}
      singular="role"
      plural="roles"
    />
  );
};
