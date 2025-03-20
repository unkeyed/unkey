"use client";
import { Tree } from "@/app/(app)/authorization/roles/[roleId]/tree";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { Button } from "@unkey/ui";
import { useEffect, useState } from "react";

export type NestedPermission = {
  id: string;
  checked: boolean;
  description: string | null;
  name: string;
  part: string;
  path: string;
  permissions: NestedPermissions;
};
export type NestedPermissions = Record<string, NestedPermission>;
export type Role = {
  id: string;
  name: string;
  keyId: string;
  active: boolean;
  description: string | null;
  nestedPermissions: NestedPermissions;
};
type PermissionTreeProps = {
  roles: Role[];
};

export function PermissionList({ roles }: PermissionTreeProps) {
  const [activeRoleId, setActiveRoleId] = useState<string | null>(
    roles.length > 0 ? roles[0].id : null,
  );
  const [key, setKey] = useState(0);

  // biome-ignore lint/correctness/useExhaustiveDependencies: Force rerender on role change
  useEffect(() => {
    // Update the key when activeRoleId changes to force a complete re-render
    setKey((prev) => prev + 1);
  }, [activeRoleId]);

  const activeRole = roles.find((r) => r.id === activeRoleId);

  return (
    <Card>
      <CardHeader className="pb-0">
        <div className="mb-2">
          <CardTitle>Permissions & Roles</CardTitle>
          <CardDescription>Connect roles with permissions to control access</CardDescription>
        </div>
        <div className="border-b border-gray-6">
          <div className="flex flex-wrap -mb-px">
            {roles.map((role) => (
              <Button
                variant="ghost"
                key={role.id}
                onClick={() => setActiveRoleId(role.id)}
                className={cn(
                  "rounded-none rounded-t inline-flex items-center px-4 py-2 text-sm font-medium border-b-2 transition-colors outline-none focus:ring-0",
                  activeRoleId === role.id
                    ? "border-primary text-primary"
                    : "border-transparent text-gray-11 hover:text-gray-12 hover:border-gray-7",
                )}
              >
                {role.name}
                {role.active && <span className="ml-2 h-2 w-2 rounded-full bg-success-9" />}
              </Button>
            ))}
          </div>
        </div>
      </CardHeader>

      <CardContent className="pt-6">
        {activeRole && activeRoleId && (
          <Tree
            key={`tree-${key}-${activeRoleId}`}
            nestedPermissions={activeRole.nestedPermissions}
            role={{ id: activeRoleId }}
          />
        )}
      </CardContent>
    </Card>
  );
}
