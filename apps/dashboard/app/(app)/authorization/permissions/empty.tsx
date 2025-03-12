"use client";

import { Empty } from "@unkey/ui";
import { useState } from "react";
import { RBACForm } from "../_components/rbac-form";

export const EmptyPermissions = () => {
  const [open, setOpen] = useState(false);
  return (
    <Empty>
      <Empty.Icon />
      <Empty.Title>No permissions found</Empty.Title>
      <Empty.Description>Create your first permission</Empty.Description>
      <Empty.Actions>
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
      </Empty.Actions>
    </Empty>
  );
};
