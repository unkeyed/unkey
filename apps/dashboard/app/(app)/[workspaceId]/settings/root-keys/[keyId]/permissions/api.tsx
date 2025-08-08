"use client";

import type { Permission } from "@unkey/db";
import { PermissionManagerCard } from "./permission-manager-card";
import { apiPermissions } from "./permissions";

type Props = {
  permissions: Permission[];
  keyId: string;
  api: {
    id: string;
    name: string;
  };
};

export const Api: React.FC<Props> = ({ keyId, api, permissions }) => {
  return (
    <PermissionManagerCard
      keyId={keyId}
      permissions={permissions}
      permissionsStructure={apiPermissions(api.id)}
      permissionManagerTitle={`${api.name}`}
      permissionManagerDescription="Permissions scoped to this API. Enabling these roles only grants access to this specific API."
    />
  );
};
