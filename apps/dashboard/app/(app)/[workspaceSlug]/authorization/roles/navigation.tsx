"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Plus, ShieldKey } from "@unkey/icons";
import dynamic from "next/dynamic";

const UpsertRoleDialog = dynamic(
  () => import("./components/upsert-role").then((mod) => mod.UpsertRoleDialog),
  {
    ssr: false,
    loading: () => (
      <NavbarActionButton disabled>
        <Plus />
        Create new role
      </NavbarActionButton>
    ),
  },
);

export function Navigation() {
  const workspace = useWorkspaceNavigation();

  return (
    <Navbar className="w-full flex justify-between">
      <Navbar.Breadcrumbs icon={<ShieldKey />} className="flex-1 w-full">
        <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/authorization/roles`}>
          Authorization
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/authorization/roles`} active>
          Roles
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <UpsertRoleDialog triggerButton />
    </Navbar>
  );
}
