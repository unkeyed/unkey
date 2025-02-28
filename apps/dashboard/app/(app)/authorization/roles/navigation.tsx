"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Badge } from "@/components/ui/badge";
import { ShieldKey } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { CreateNewRole } from "./create-new-role";
interface NavigationProps {
  workspace: {
    roles: Array<{
      id: string;
      name: string;
      description: string | null;
      keys: Array<{
        key: {
          deletedAtM: number | null;
        };
      }>;
      permissions: Array<{
        permission: any; // We could type this further if needed
      }>;
    }>;
    permissions: Array<any>; // We need this for CreateNewRole component
  };
}

export function Navigation({ workspace }: NavigationProps) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<ShieldKey />}>
        <Navbar.Breadcrumbs.Link href="/authorization/roles">Authorization</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href="/authorization/roles" active>
          Roles
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <Badge variant="secondary" className="h-8">
          {Intl.NumberFormat().format(workspace.roles.length)} /{" "}
          {Intl.NumberFormat().format(Number.POSITIVE_INFINITY)} used{" "}
        </Badge>
        <CreateNewRole
          trigger={<Button variant="primary">Create New Role</Button>}
          permissions={workspace.permissions}
        />
      </Navbar.Actions>
    </Navbar>
  );
}
