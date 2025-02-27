"use client";

import { Navbar } from "@/components/navbar";
import { ShieldKey } from "@unkey/icons";
import { Badge } from "@/components/ui/badge";
import { Button } from "@unkey/ui";
import { CreateNewPermission } from "./create-new-permission";

// Reusable for settings where we only change the link
export function Navigation({ numberOfPermissions }: { numberOfPermissions: number }) {
  return (
    <Navbar>
        <Navbar.Breadcrumbs icon={<ShieldKey />}>
          <Navbar.Breadcrumbs.Link href="/authorization/roles">
            Authorization
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="/authorization/permissions" active>
            Permissions
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge variant="secondary" className="h-8">
            {Intl.NumberFormat().format(numberOfPermissions)} /{" "}
            {Intl.NumberFormat().format(Number.POSITIVE_INFINITY)} used{" "}
          </Badge>
          <CreateNewPermission trigger={<Button variant="primary">Create New Permission</Button>} />
        </Navbar.Actions>
      </Navbar>
  );
}