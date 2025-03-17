"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { formatNumber } from "@/lib/fmt";
import { ShieldKey } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { RBACForm } from "../_components/rbac-form";

export function Navigation({
  numberOfPermissions,
}: {
  numberOfPermissions: number;
}) {
  const [open, setOpen] = useState(false);
  return (
    <>
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
          <Button
            disabled
            variant="outline"
            size="md"
            className="text-xs font-medium ph-no-capture h-8 bg-grayA-3 hover:bg-grayA-3 !text-grayA-8"
          >
            <div className="flex gap-1 items-center justify-center text-sm">
              {formatNumber(numberOfPermissions)} Permissions
            </div>
          </Button>
          <NavbarActionButton
            title="Click to create a new permission"
            onClick={() => setOpen(true)}
          >
            Create New Permission
          </NavbarActionButton>
        </Navbar.Actions>
      </Navbar>
      <RBACForm
        isModalOpen={open}
        onOpenChange={setOpen}
        title="Create a new permission"
        buttonText="Create New Permission"
        footerText="Permissions allow your key to do certain actions"
        formId="permission-form"
        type="create"
        itemType="permission"
      />
    </>
  );
}
