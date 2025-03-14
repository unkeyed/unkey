"use client";
import { RBACForm } from "@/app/(app)/authorization/_components/rbac-form";
import type { Permission } from "@unkey/db";
import { Button } from "@unkey/ui";
import { useState } from "react";

interface RBACButtonsProps {
  permissions?: Permission[];
}

export function RBACButtons({ permissions = [] }: RBACButtonsProps) {
  const [isCreateRoleModalOpen, setIsCreateRoleModalOpen] = useState(false);
  const [isCreatePermissionModalOpen, setIsCreatePermissionModalOpen] = useState(false);

  const permissionIds = permissions.map((permission) => permission.id);

  return (
    <div className="flex items-center gap-2 border-border">
      <RBACForm
        isModalOpen={isCreateRoleModalOpen}
        onOpenChange={setIsCreateRoleModalOpen}
        title="Create New Role"
        description="Roles are used to group permissions together and are attached to keys."
        buttonText="Create Role"
        formId="create-role-form"
        type="create"
        itemType="role"
        additionalParams={{
          permissionIds: permissionIds,
        }}
      >
        <Button size="md" variant="outline" onClick={() => setIsCreateRoleModalOpen(true)}>
          Create New Role
        </Button>
      </RBACForm>

      <RBACForm
        isModalOpen={isCreatePermissionModalOpen}
        onOpenChange={setIsCreatePermissionModalOpen}
        title="Create New Permission"
        description="Permissions define what actions can be performed by API keys with specific roles."
        buttonText="Create Permission"
        formId="create-permission-form"
        type="create"
        itemType="permission"
      >
        <Button size="md" variant="outline" onClick={() => setIsCreatePermissionModalOpen(true)}>
          Create New Permission
        </Button>
      </RBACForm>
    </div>
  );
}
