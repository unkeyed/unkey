import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { UpsertRoleDialog } from "../../../../upsert-role";
import { useFetchConnectedKeysAndPermsData } from "./hooks/use-fetch-connected-keys-and-perms";

export const EditRole = ({
  role,
  isOpen,
  onClose,
  shouldFetch,
}: {
  role: RoleBasic;
  isOpen: boolean;
  onClose: () => void;
  shouldFetch: boolean;
}) => {
  const { permissions, keys } = useFetchConnectedKeysAndPermsData(role.roleId, shouldFetch);

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
