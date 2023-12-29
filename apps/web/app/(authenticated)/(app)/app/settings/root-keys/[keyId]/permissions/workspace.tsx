"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Role, apiActions } from "@unkey/rbac";
import { useState } from "react";
import { Permission } from "./permission";
import { workspaceRoles } from "./roles";

type Props = {
  permissions: string[];
  keyId: string;
};

export const Workspace: React.FC<Props> = ({ keyId, permissions }) => {
  const [expanded, setExpanded] = useState(false);

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between">
        <div className="flex flex-col space-y-1.5">
          <CardTitle>Workspace</CardTitle>
        </div>
        <Button size="sm" variant="outline" onClick={() => setExpanded(!expanded)}>
          {expanded ? "Hide disabled roles" : "Show all"}
        </Button>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col gap-1">
          {Object.entries(workspaceRoles).map(([category, roles]) => {
            return (
              <div className="flex items-start gap-8">
                <span className="font-semibold capitalize">{category}</span>
                <ol className="flex flex-col gap-1">
                  {Object.entries(roles).map(([action, { description, role }]) => (
                    <Permission
                      key={action}
                      rootKeyId={keyId}
                      role={role}
                      label={action}
                      description={description}
                      checked={permissions.includes(action)}
                    />
                  ))}
                </ol>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
};
