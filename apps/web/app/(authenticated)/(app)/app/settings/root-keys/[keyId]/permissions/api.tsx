"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Permission } from "./permission";
import { apiRoles } from "./roles";

type Props = {
  permissions: string[];
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
          {Object.entries(apiRoles(api.id)).map(([category, roles]) => {
            return (
              <div className="flex flex-col gap-2">
                <span className="font-medium">{category}</span>
                <div className="flex flex-col gap-1">
                  {Object.entries(roles).map(([action, { description, role }]) => {
                    return (
                      <Permission
                        key={action}
                        rootKeyId={keyId}
                        role={role}
                        label={action}
                        description={description}
                        checked={permissions.includes(api.id)}
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
