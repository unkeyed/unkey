"use client";
import { CopyButton } from "@/components/dashboard/copy-button";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { Badge } from "@/components/ui/badge";
import type { Permission } from "@unkey/db";
import { ShieldKey } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { DeletePermission } from "./delete-permission";
// Reusable for settings where we only change the link
export function Navigation({
  permissionId,
  permission,
}: {
  permissionId: string;
  permission: Permission;
}) {
  const shouldShowTooltip = permission.name.length > 16;

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<ShieldKey />}>
        <Navbar.Breadcrumbs.Link href="/authorization/roles">Authorization</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href="/authorization/permissions">
          Permissions
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/authorization/permissions/${permissionId}`}
          isIdentifier
          active
        >
          {permissionId}
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <Badge
          key="permission-name"
          variant="secondary"
          className="w-40 font-mono font-medium ph-no-capture h-8"
        >
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="flex items-center justify-between gap-2 w-full truncate">
                <span className="truncate">{permission.name}</span>
                <div>
                  <CopyButton value={permission.name} />
                </div>
              </div>
            </TooltipTrigger>
            {shouldShowTooltip && (
              <TooltipContent>
                <span className="text-xs font-medium">{permission.name}</span>
              </TooltipContent>
            )}
          </Tooltip>
        </Badge>
        <Badge
          key="permission-id"
          variant="secondary"
          className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture h-8"
        >
          {permission.id}
          <CopyButton value={permission.id} />
        </Badge>
        <DeletePermission
          key="delete-permission"
          trigger={
            <NavbarActionButton variant="destructive" color="danger" className="">
              Delete Permission
            </NavbarActionButton>
          }
          permission={permission}
        />{" "}
      </Navbar.Actions>
    </Navbar>
  );
}
