"use client";

import type { Permission } from "@unkey/db";
import { PermissionManagerCard } from "./permission-manager-card";
import { workspacePermissions } from "./permissions";

type Props = {
  permissions: Permission[];
  keyId: string;
};

export const Workspace: React.FC<Props> = ({ keyId, permissions }) => {
  return (
    <PermissionManagerCard
      keyId={keyId}
      permissions={permissions}
      permissionsStructure={workspacePermissions}
      permissionManagerTitle="Workspace Permissions"
      permissionManagerDescription="Manage workspace permissions"
    />
  );
};
