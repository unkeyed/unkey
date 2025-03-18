"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import type { Role } from "@unkey/db";
import { ShieldKey } from "@unkey/icons";
import { useState } from "react";
import { RBACForm } from "../../_components/rbac-form";
import { DeleteRole } from "./delete-role";

export function Navigation({ role }: { role: Role }) {
  const [isUpdateModalOpen, setIsUpdateModalOpen] = useState(false);

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<ShieldKey />}>
        <Navbar.Breadcrumbs.Link href="/authorization/roles">Authorization</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href="/authorization/roles">Roles</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/authorization/roles/${role.id}`} isIdentifier active>
          {role.id}
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <NavbarActionButton onClick={() => setIsUpdateModalOpen(true)}>
          Update Role
        </NavbarActionButton>

        <DeleteRole
          role={role}
          trigger={
            <NavbarActionButton variant="destructive" color="danger" className="">
              Delete Role
            </NavbarActionButton>
          }
        />
        <RBACForm
          isModalOpen={isUpdateModalOpen}
          onOpenChange={setIsUpdateModalOpen}
          title="Update Role"
          description="Roles are used to group permissions together and are attached to keys."
          buttonText="Save"
          formId="update-role-form"
          type="update"
          itemType="role"
          item={{
            id: role.id,
            name: role.name,
            description: role.description,
          }}
        />
      </Navbar.Actions>
    </Navbar>
  );
}
