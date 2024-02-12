"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Permission } from "@unkey/db";
import { PermissionToggle } from "./permission_toggle";
import { apiPermissions } from "./permissions";

type Props = {
  permissions: Permission[];
  keyId: string;
  api: {
    id: string;
    name: string;
  };
};

export const Api: React.FC<Props> = ({ keyId, api, permissions }) => {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{api.name}</CardTitle>
        <CardDescription>
          Permissions scoped to this API. Enabling these roles only grants access to this specific
          API.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col gap-4">
          {Object.entries(apiPermissions(api.id)).map(([category, roles]) => {
            return (
              <div className="flex flex-col gap-2">
                <span className="font-medium">{category}</span>
                <div className="flex flex-col gap-1">
                  {Object.entries(roles).map(([action, { description, permission }]) => {
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
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
};
