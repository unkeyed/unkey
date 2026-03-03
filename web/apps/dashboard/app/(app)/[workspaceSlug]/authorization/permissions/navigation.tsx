"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Plus, ShieldKey } from "@unkey/icons";
import dynamic from "next/dynamic";
import { useState } from "react";

const UpsertPermissionDialog = dynamic(
  () => import("./components/upsert-permission").then((mod) => mod.UpsertPermissionDialog),
  { ssr: false },
);

export function Navigation() {
  const workspace = useWorkspaceNavigation();
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <Navbar className="w-full flex justify-between">
        <Navbar.Breadcrumbs icon={<ShieldKey />} className="flex-1 w-full">
          <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/authorization/roles`}>
            Authorization
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/authorization/permissions`} active>
            Permissions
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <NavbarActionButton onClick={() => setIsOpen(true)}>
            <Plus />
            Create new permission
          </NavbarActionButton>
        </Navbar.Actions>
      </Navbar>
      <UpsertPermissionDialog isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </>
  );
}
