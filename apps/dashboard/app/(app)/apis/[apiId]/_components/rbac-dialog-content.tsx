"use client";

import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { Badge, Button, Loading } from "@unkey/ui";
import dynamic from "next/dynamic";

const PermissionList = dynamic(
  () =>
    import("../keys/[keyAuthId]/[keyId]/components/rbac/permissions").then((mod) => ({
      default: mod.PermissionList,
    })),
  { ssr: false },
);

const RBACButtons = dynamic(
  () =>
    import("../keys/[keyAuthId]/[keyId]/components/rbac/rbac-buttons").then((mod) => ({
      default: mod.RBACButtons,
    })),
  { ssr: false },
);

type Props = {
  keyId: string;
  keyspaceId: string;
};

export function RBACDialogContent({ keyId, keyspaceId }: Props) {
  const trpcUtils = trpc.useUtils();

  const {
    data: permissionsData,
    isLoading,
    isRefetching,
    error,
  } = trpc.key.fetchPermissions.useQuery({
    keyId,
    keyspaceId,
  });

  const { transientPermissionIds, rolesList } = calculatePermissionData(permissionsData);

  if (isLoading) {
    return (
      <div className="flex justify-center items-center p-4 min-h-[250px] [&_svg]:size-10">
        <Loading size={18} />
      </div>
    );
  }

  if (error || !permissionsData) {
    return (
      <div className="flex flex-col items-center justify-center p-8 gap-4 min-h-[250px]">
        <div className="text-accent-10 text-sm">Could not retrieve permission data</div>
        <div className="text-accent-10 text-xs max-w-[400px] text-center">
          There was an error loading the permissions for this key. Please try again or contact
          support if the issue persists.
        </div>
        <Button
          variant="primary"
          size="xlg"
          className="mt-2 w-[200px] h-9 rounded-md focus:ring-4 focus:ring-accent-9 focus:ring-offset-2"
          loading={isRefetching}
          onClick={() => {
            // Refetch permissions data
            if (keyId && keyspaceId) {
              trpcUtils.key.fetchPermissions.invalidate({
                keyId,
                keyspaceId,
              });
            }
          }}
        >
          Try again
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex w-full flex-1 items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Badge variant="secondary" className="h-8">
            {formatNumber(permissionsData.roles.length)} Roles{" "}
          </Badge>
          <Badge variant="secondary" className="h-8">
            {formatNumber(transientPermissionIds.size)} Permissions
          </Badge>
        </div>
        <RBACButtons permissions={permissionsData.workspace.permissions.roles} />
      </div>
      <div className="min-h-[250px]">
        {keyId ? (
          <PermissionList roles={rolesList} keyId={keyId} />
        ) : (
          <div className="flex justify-center items-center p-4">
            <div className="text-accent-10 text-sm">No key selected</div>
          </div>
        )}
      </div>
    </div>
  );
}

type WorkspaceRole = {
  id: string;
  name: string;
  permissions: { permissionId: string }[];
};

type PermissionsResponse = {
  roles: { roleId: string }[];
  workspace: { roles: WorkspaceRole[]; permissions: { roles: unknown } };
};

function calculatePermissionData(permissionsData?: PermissionsResponse) {
  const transientPermissionIds = new Set<string>();
  const rolesList: { id: string; name: string; isActive: boolean }[] = [];

  if (!permissionsData) {
    return { transientPermissionIds, rolesList };
  }

  // Mimic the original implementation logic
  const connectedRoleIds = new Set<string>();

  for (const role of permissionsData.roles) {
    connectedRoleIds.add(role.roleId);
  }

  for (const role of permissionsData.workspace.roles) {
    if (connectedRoleIds.has(role.id)) {
      for (const p of role.permissions) {
        transientPermissionIds.add(p.permissionId);
      }
    }
  }

  // Build roles list matching the original format
  const roles = permissionsData.workspace.roles.map((role: { id: string; name: string }) => {
    return {
      id: role.id,
      name: role.name,
      isActive: permissionsData.roles.some(
        (keyRole: { roleId: string }) => keyRole.roleId === role.id,
      ),
    };
  });

  return {
    transientPermissionIds,
    rolesList: roles,
  };
}
