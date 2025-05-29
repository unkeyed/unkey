"use client";

import { QuickNavPopover } from "@/components/navbar-popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { Navbar } from "@/components/navigation/navbar";
import { Badge } from "@/components/ui/badge";
import { useIsMobile } from "@/hooks/use-mobile";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { ChevronExpandY, Gauge, Gear, Plus, ShieldKey } from "@unkey/icons";
import { AnimatedLoadingSpinner, Button } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useState } from "react";
import { PermissionList } from "./keys/[keyAuthId]/[keyId]/components/rbac/permissions";
import { RBACButtons } from "./keys/[keyAuthId]/[keyId]/components/rbac/rbac-buttons";
import { getKeysTableActionItems } from "./keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover.constants";

const CreateKeyDialog = dynamic(
  () =>
    import("./_components/create-key").then((mod) => ({
      default: mod.CreateKeyDialog,
    })),
  {
    ssr: false,
    loading: () => (
      <NavbarActionButton disabled>
        <Plus />
        Create new key
      </NavbarActionButton>
    ),
  },
);

const DialogContainer = dynamic(
  () => import("@/components/dialog-container").then((mod) => mod.DialogContainer),
  {
    ssr: false,
  },
);

const KeysTableActionPopover = dynamic(
  () =>
    import(
      "./keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover"
    ).then((mod) => ({ default: mod.KeysTableActionPopover })),
  {
    ssr: false,
    loading: () => (
      <NavbarActionButton disabled>
        <Gear size="sm-regular" />
        Settings
      </NavbarActionButton>
    ),
  },
);

export const ApisNavbar = ({
  api,
  apis,
  activePage,
  keyData,
}: {
  api: {
    id: string;
    name: string;
    keyAuthId: string | null;
    keyspaceDefaults: {
      prefix?: string;
      bytes?: number;
    } | null;
  };
  apis: {
    id: string;
    name: string;
  }[];
  activePage?: {
    href: string;
    text: string;
  };
  keyData?: KeyDetails | null;
}) => {
  const isMobile = useIsMobile();
  const trpcUtils = trpc.useUtils();
  const [showRBAC, setShowRBAC] = useState(false);

  const keyId = keyData?.id || "";
  const keyspaceId = api.keyAuthId || "";
  const shouldFetchPermissions = Boolean(keyId) && Boolean(keyspaceId);

  const {
    data: permissionsData,
    isLoading,
    isRefetching,
    error,
  } = trpc.key.fetchPermissions.useQuery(
    {
      keyId,
      keyspaceId,
    },
    {
      enabled: shouldFetchPermissions,
    },
  );

  const { transientPermissionIds, rolesList } = calculatePermissionData(permissionsData);

  return (
    <>
      <div className="w-full">
        <Navbar className="w-full flex justify-between">
          <Navbar.Breadcrumbs className="flex-1 w-full" icon={<Gauge />}>
            {!isMobile && (
              <>
                <Navbar.Breadcrumbs.Link href="/apis" className="max-md:hidden">
                  APIs
                </Navbar.Breadcrumbs.Link>

                <Navbar.Breadcrumbs.Link
                  href={`/apis/${api.id}`}
                  isIdentifier
                  className="group max-md:hidden"
                  noop
                >
                  <QuickNavPopover
                    items={apis.map((api) => ({
                      id: api.id,
                      label: api.name,
                      href: `/apis/${api.id}`,
                    }))}
                    shortcutKey="N"
                  >
                    <div className="text-accent-10 group-hover:text-accent-12">{api.name}</div>
                  </QuickNavPopover>
                </Navbar.Breadcrumbs.Link>
              </>
            )}

            <Navbar.Breadcrumbs.Link href={activePage?.href ?? ""} noop active={!keyData}>
              <QuickNavPopover
                items={[
                  {
                    id: "requests",
                    label: "Requests",
                    href: `/apis/${api.id}`,
                  },
                  {
                    id: "keys",
                    label: "Keys",
                    href: `/apis/${api.id}/keys/${api.keyAuthId}`,
                  },
                  {
                    id: "settings",
                    label: "Settings",
                    href: `/apis/${api.id}/settings`,
                  },
                ]}
              >
                <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
                  {activePage?.text ?? ""}
                  <ChevronExpandY className="size-4" />
                </div>
              </QuickNavPopover>
            </Navbar.Breadcrumbs.Link>
            {keyData && (
              <Navbar.Breadcrumbs.Link
                href={`/apis/${api.id}/keys/${api.keyAuthId}/${keyData.id}`}
                className="max-md:hidden"
                isLast
                isIdentifier
                active
              >
                {keyData.id?.substring(0, 8)}...
                {keyData.id?.substring(keyData.id?.length - 4)}
              </Navbar.Breadcrumbs.Link>
            )}
          </Navbar.Breadcrumbs>
          {keyData?.id ? (
            <div className="flex gap-3 items-center">
              <Navbar.Actions>
                <NavbarActionButton
                  onClick={() => setShowRBAC(true)}
                  disabled={!shouldFetchPermissions}
                >
                  <ShieldKey size="sm-regular" />
                  Permissions
                </NavbarActionButton>
              </Navbar.Actions>
              <Navbar.Actions>
                <KeysTableActionPopover items={getKeysTableActionItems(keyData)}>
                  <NavbarActionButton>
                    <Gear size="sm-regular" />
                    Settings
                  </NavbarActionButton>
                </KeysTableActionPopover>
                <CopyableIDButton value={keyData.id} />
              </Navbar.Actions>
            </div>
          ) : (
            api.keyAuthId && (
              <CreateKeyDialog
                keyspaceId={api.keyAuthId}
                apiId={api.id}
                keyspaceDefaults={api.keyspaceDefaults}
              />
            )
          )}
        </Navbar>
      </div>
      <DialogContainer
        isOpen={showRBAC}
        onOpenChange={() => setShowRBAC(false)}
        title="Key Permissions & Roles"
        subTitle="Manage access control for this API key with role-based permissions"
        className="max-w-[800px] max-h-[90vh] overflow-y-auto"
      >
        {isLoading ? (
          <div className="flex justify-center items-center p-4 min-h-[250px] [&_svg]:size-10">
            <AnimatedLoadingSpinner />
          </div>
        ) : error || !permissionsData ? (
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
        ) : (
          <div className="flex flex-col gap-4 ">
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
              {/* Only render PermissionList if we have a valid keyId */}
              {keyId ? (
                <PermissionList roles={rolesList} keyId={keyId} />
              ) : (
                <div className="flex justify-center items-center p-4">
                  <div className="text-accent-10 text-sm">No key selected</div>
                </div>
              )}
            </div>
          </div>
        )}
      </DialogContainer>
    </>
  );
};

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
