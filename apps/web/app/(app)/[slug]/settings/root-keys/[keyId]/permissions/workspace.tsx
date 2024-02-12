"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Permission } from "@unkey/db";
import { PermissionToggle } from "./permission_toggle";
import { workspacePermissions } from "./permissions";

type Props = {
  permissions: Permission[];
  keyId: string;
};

export const Workspace: React.FC<Props> = ({ keyId, permissions }) => {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Workspace</CardTitle>
        <CardDescription>Manage workspace permissions</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col gap-4">
          {Object.entries(workspacePermissions).map(([category, allPermissions]) => (
            <div className="flex flex-col gap-2">
              <span className="font-medium">{category}</span>{" "}
              <div className="flex flex-col gap-1">
                {Object.entries(allPermissions).map(([action, { description, permission }]) => {
                  return (
                    <PermissionToggle
                      key={action}
                      rootKeyId={keyId}
                      permissionName={permission}
                      label={action}
                      description={description}
                      checked={permissions.some((p) => p.name === permission)}
                    />
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
};
