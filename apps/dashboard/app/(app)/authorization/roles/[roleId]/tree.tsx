"use client";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ChevronRight } from "lucide-react";
import type React from "react";
import { useEffect, useState } from "react";
import { PermissionToggle } from "./permission-toggle";

export const revalidate = 0;

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

type Props = {
  nestedPermissions: NestedPermissions;
  role: {
    id: string;
  };
};

export const Tree: React.FC<Props> = ({ nestedPermissions, role }) => {
  const [openAll, setOpenAll] = useState(false);

  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between">
        <div className="flex flex-col space-y-1.5 ">
          <CardTitle>Permissions for role</CardTitle>
          <CardDescription>Add or remove permissions from this role</CardDescription>
        </div>
        <div className="flex items-center gap-2">
          <Label>{openAll ? "Collapse" : "Expand"} All</Label>
          <Switch checked={openAll} onCheckedChange={setOpenAll} />
        </div>
      </CardHeader>

      <CardContent className="flex flex-col gap-1 ">
        {Object.entries(nestedPermissions).map(([k, p]) => (
          <RecursivePermission k={k} {...p} roleId={role.id} openAll={openAll} />
        ))}
      </CardContent>
    </Card>
  );
};

export const RecursivePermission: React.FC<
  NestedPermission & { k: string; roleId: string; openAll: boolean }
> = ({ k, openAll, id, name, permissions, roleId, checked, description }) => {
  const [open, setOpen] = useState(openAll);
  useEffect(() => {
    setOpen(openAll);
  }, [openAll]);
  const children = Object.values(permissions);

  if (children.length === 0) {
    return (
      <Tooltip delayDuration={50}>
        <TooltipTrigger className="flex flex-col items-start px-2 py-1 ml-4 hover:bg-secondary">
          <div className="flex items-center gap-2 ">
            <PermissionToggle permissionId={id} roleId={roleId} checked={checked} />
            <pre className="text-sm text-content">{k}</pre>
          </div>
          <p className="ml-6 text-xs text-content-subtle">{description}</p>
        </TooltipTrigger>
        <TooltipContent side="top" align="start" avoidCollisions={true}>
          <div className="flex items-center justify-start max-w-sm gap-2 text-content">
            <pre className="text-ellipsis overflow-hidden hover:overflow-visible">{name}</pre>
            <div>
              <CopyButton value={name} />
            </div>
          </div>
        </TooltipContent>
      </Tooltip>
    );
  }
  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger className="flex items-center gap-1 transition-all [&[data-state=open]>svg]:rotate-90 ">
        <ChevronRight className="w-4 h-4 transition-transform duration-200" />
        <pre className="text-sm">{k}</pre>
      </CollapsibleTrigger>

      <CollapsibleContent className="flex flex-col gap-1 ml-2 border-l border-border ">
        {Object.entries(permissions).map(([k2, p]) => (
          <RecursivePermission key={p.id} k={k2} {...p} roleId={roleId} openAll={openAll} />
        ))}
      </CollapsibleContent>
    </Collapsible>
  );
};
