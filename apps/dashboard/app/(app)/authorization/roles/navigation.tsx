"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { formatNumber } from "@/lib/fmt";
import { ShieldKey } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { RBACForm } from "../_components/rbac-form";

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
    permissions: Array<any>; // Kept for reference
  };
}

export function Navigation({ workspace }: NavigationProps) {
  const [isRoleModalOpen, setIsRoleModalOpen] = useState(false);

  return (
    <>
      <Navbar>
        <Navbar.Breadcrumbs icon={<ShieldKey />}>
          <Navbar.Breadcrumbs.Link href="/authorization/roles">
            Authorization
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="/authorization/permissions" active>
            Roles
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Button
            disabled
            variant="outline"
            size="md"
            className="text-xs font-medium ph-no-capture h-8 bg-grayA-3 hover:bg-grayA-3 !text-grayA-8"
          >
            <div className="flex gap-1 items-center justify-center text-sm">
              {formatNumber(workspace.roles.length)} Roles
            </div>
          </Button>
          <NavbarActionButton
            title="Click to create a new role"
            onClick={() => setIsRoleModalOpen(true)}
          >
            Create New Role
          </NavbarActionButton>
        </Navbar.Actions>
      </Navbar>

      <RBACForm
        isModalOpen={isRoleModalOpen}
        onOpenChange={setIsRoleModalOpen}
        title="Create a new role"
        buttonText="Create New Role"
        footerText="Roles group permissions together"
        formId="role-form"
        type="create"
        itemType="role"
        additionalParams={{ permissionIds: [] }}
      />
    </>
  );
}
