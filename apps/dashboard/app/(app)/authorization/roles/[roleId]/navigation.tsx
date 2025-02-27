"use client";

import { Navbar } from "@/components/navbar";
import { ShieldKey } from "@unkey/icons";
import { DeleteRole } from "./delete-role";
import { UpdateRole } from "./update-role";
import { Button } from "@unkey/ui";
import type { Role } from "@unkey/db";

export function Navigation({role}: {role: Role} ) {
  return (
    <Navbar>
        <Navbar.Breadcrumbs icon={<ShieldKey />}>
          <Navbar.Breadcrumbs.Link href="/authorization/roles">
            Authorization
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="/authorization/roles">Roles</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/authorization/permissions/${role.id}`}
            isIdentifier
            active
          >
            {role.id}
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <UpdateRole role={role} trigger={<Button>Update Role</Button>} />
          <DeleteRole role={role} trigger={<Button variant="destructive">Delete Role</Button>} />
        </Navbar.Actions>
      </Navbar>
  );
}