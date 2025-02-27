"use client";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/navbar";
import { ShieldKey } from "@unkey/icons";
import { Badge } from "@/components/ui/badge";
import { Button, TooltipTrigger, Tooltip, TooltipContent } from "@unkey/ui";
import { DeletePermission } from "./delete-permission";
import type { Permission } from "@unkey/db";
// Reusable for settings where we only change the link
export function Navigation({ permissionId, permission }: { permissionId: string, permission: Permission}) {
 const shouldShowTooltip = permission.name.length > 16;

  return (
    <Navbar>
        <Navbar.Breadcrumbs icon={<ShieldKey />}>
          <Navbar.Breadcrumbs.Link href="/authorization/roles">
            Authorization
          </Navbar.Breadcrumbs.Link>
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
            className="w-40 font-mono font-medium ph-no-capture"
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
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {permission.id}
            <CopyButton value={permission.id} />
          </Badge>
          <DeletePermission
            key="delete-permission"
            trigger={<Button variant="destructive">Delete</Button>}
            permission={permission}
          />{" "}
        </Navbar.Actions>
      </Navbar>
  );
}