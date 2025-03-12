"use client";

import { Empty } from "@unkey/ui";
import { useState } from "react";
import { RBACForm } from "../_components/rbac-form";

export const EmptyRoles = () => {
  const [isRoleModalOpen, setIsRoleModalOpen] = useState(false);
  return (
    <Empty>
      <Empty.Icon />
      <Empty.Title>No roles found</Empty.Title>
      <Empty.Description>Create your first role</Empty.Description>
      <Empty.Actions>
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
      </Empty.Actions>
    </Empty>
  );
};
