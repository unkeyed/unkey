"use client";
import { Plus } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useState } from "react";
import { RBACForm } from "../_components/rbac-form";

export const EmptyPermissions = () => {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Empty className="max-w-2xl mx-auto">
        <Empty.Icon />
        <Empty.Title>No permissions found</Empty.Title>
        <Empty.Description>
          Permissions define specific actions that API keys can perform. <br />
          Add permissions to build granular access control for your resources.
        </Empty.Description>
        <Empty.Actions>
          <Button size="md" color="default" onClick={() => setOpen(true)}>
            <Plus />
            Create New Permission
          </Button>
        </Empty.Actions>
      </Empty>
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
};
