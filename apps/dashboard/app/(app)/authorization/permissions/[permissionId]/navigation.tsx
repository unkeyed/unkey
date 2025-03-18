"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { Navbar } from "@/components/navigation/navbar";
import type { Permission } from "@unkey/db";
import { ShieldKey } from "@unkey/icons";
import { DeletePermission } from "./delete-permission";
// Reusable for settings where we only change the link
export function Navigation({
  permissionId,
  permission,
}: {
  permissionId: string;
  permission: Permission;
}) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<ShieldKey />}>
        <Navbar.Breadcrumbs.Link href="/authorization/permissions">
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
        <CopyableIDButton value={permission.name} />
        <CopyableIDButton value={permission.id} />
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
