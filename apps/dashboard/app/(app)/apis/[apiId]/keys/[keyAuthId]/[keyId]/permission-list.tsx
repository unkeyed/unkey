"use client";
import { RecursivePermission } from "@/app/(app)/authorization/roles/[roleId]/tree";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ChevronRight } from "lucide-react";
import { useEffect, useState } from "react";
import { RoleToggle } from "./role-toggle";

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

export default function PermissionTree({ roles }: PermissionTreeProps) {
  const [openAll, setOpenAll] = useState(false);
  const [openRoles, setOpenRoles] = useState<string[]>([]);

  useEffect(() => {
    setOpenRoles(openAll ? roles.map((role) => role.id) : []);
  }, [openAll, roles]);

  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between">
        <div className="flex flex-col space-y-1.5">
          <CardTitle>Permissions</CardTitle>
        </div>
        <div className="flex items-center gap-2">
          <Label>{openAll ? "Collapse" : "Expand"} All Roles</Label>
          <Switch checked={openAll} onCheckedChange={setOpenAll} />
        </div>
      </CardHeader>

      <CardContent className="flex flex-col gap-4">
        {roles.map((role) => {
          const isOpen = openRoles.includes(role.id);
          return (
            <Collapsible
              key={role.id}
              open={isOpen}
              onOpenChange={(open) => {
                setOpenRoles((prev) =>
                  open
                    ? prev.includes(role.id)
                      ? prev
                      : [...prev, role.id]
                    : prev.filter((id) => id !== role.id),
                );
              }}
            >
              <CollapsibleTrigger className="flex items-center gap-1 transition-all [&[data-state=open]>svg]:rotate-90 ">
                <Tooltip delayDuration={50}>
                  <TooltipTrigger className="flex items-center gap-2">
                    <ChevronRight className="w-4 h-4 transition-transform duration-200" />
                    <RoleToggle keyId={role.keyId} roleId={role.id} checked={role.active} />
                    <span className="text-sm">{role.name}</span>
                  </TooltipTrigger>
                  <TooltipContent side="top" align="start" avoidCollisions={true}>
                    <div className="flex items-center justify-start max-w-sm gap-2 text-content">
                      <span className="text-ellipsis overflow-hidden hover:overflow-visible">
                        {role.name}
                      </span>
                      <div>
                        <CopyButton value={role.name} />
                      </div>
                    </div>
                  </TooltipContent>
                </Tooltip>
              </CollapsibleTrigger>
              <CollapsibleContent className="pt-2 pb-4">
                <div className="flex flex-col gap-1 ml-4">
                  {Object.entries(role.nestedPermissions).map(([k, p]) => (
                    <RecursivePermission
                      key={p.id}
                      k={k}
                      {...p}
                      roleId={role.id}
                      openAll={openAll}
                    />
                  ))}
                </div>
              </CollapsibleContent>
            </Collapsible>
          );
        })}
      </CardContent>
    </Card>
  );
}
