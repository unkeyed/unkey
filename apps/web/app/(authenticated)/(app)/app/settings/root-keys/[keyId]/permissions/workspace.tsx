"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Permission } from "./permission";
import { workspaceRoles } from "./roles";

type Props = {
  permissions: string[];
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
        <div className="flex flex-col gap-1">
          {Object.entries(workspaceRoles).map(([action, { description, role }]) => {
            return (
              <Permission
                key={action}
                rootKeyId={keyId}
                role={role}
                label={action}
                description={description}
                checked={permissions.includes(role)}
              />
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
};
