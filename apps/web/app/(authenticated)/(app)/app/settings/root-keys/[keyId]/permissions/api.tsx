"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { apiActions } from "@unkey/rbac";
import { useState } from "react";
import { Permission } from "./permission";
import { apiRoles } from "./roles";

type Props = {
  api: {
    id: string;
    name: string;
  };
  permissions: string[];
  keyId: string;
};

type PermissionTree = {
  [key: string]: PermissionTree | boolean;
};

function buildPermissionTree(permissions: string[]): PermissionTree {
  const tree: PermissionTree = {};

  for (const permission of permissions) {
    console.log({ permission });
    addToTree(tree, permission.split("."));
  }

  return tree;
}

function addToTree(node: PermissionTree, parts: string[]): void {
  const [currentPart, ...remainingParts] = parts;
  console.log({ currentPart, remainingParts });
  const isLeaf = remainingParts.length === 0;
  node[currentPart] ??= isLeaf ? true : {};

  if (!isLeaf) {
    addToTree(node[currentPart] as PermissionTree, remainingParts);
  }
}
export const Api: React.FC<Props> = ({ api, keyId, permissions }) => {
  const [expanded, setExpanded] = useState(false);

  const roleTree = buildPermissionTree(Object.values(apiRoles).map((r) => r.role(api.id)));
  console.log(JSON.stringify({ roleTree }, null, 2));
  return (
    <Card key={api.id}>
      <CardHeader className="flex flex-row items-start justify-between">
        <div className="flex flex-col space-y-1.5">
          <CardTitle>{api.name}</CardTitle>
          <CardDescription>{api.id}</CardDescription>
        </div>
        <Button size="sm" variant="outline" onClick={() => setExpanded(!expanded)}>
          {expanded ? "Hide disabled roles" : "Show all"}
        </Button>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col gap-1">
          {Object.entries(apiRoles).map(([category, roles]) => {
            return (
              <div className="flex items-start gap-8">
                <span className="font-semibold capitalize w-1/4 text-right">{category}</span>
                <ol className="flex flex-col gap-1 w-3/4">
                  {Object.entries(roles).map(([action, { description, role }]) => (
                    <Permission
                      key={action}
                      rootKeyId={keyId}
                      role={typeof role === "string" ? role : role(api.id)}
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
