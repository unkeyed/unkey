import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { UpsertPermissionDialog } from "../../../../upsert-permission";

export const EditPermission = ({
  permission,
  isOpen,
  onClose,
}: {
  permission: Permission;
  isOpen: boolean;
  onClose: () => void;
}) => {
  return (
    <UpsertPermissionDialog
      existingPermission={{
        id: permission.permissionId,
        name: permission.name,
        slug: permission.slug,
        description: permission.description,
      }}
      isOpen={isOpen}
      onClose={onClose}
    />
  );
};
