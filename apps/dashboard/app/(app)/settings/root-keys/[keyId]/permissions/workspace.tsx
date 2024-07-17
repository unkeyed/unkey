"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { Permission } from "@unkey/db";
import { PermissionToggle } from "./permission_toggle";
import { workspacePermissions } from "./permissions";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Label } from "@/components/ui/label";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Badge, Link, MoreHorizontal, MoreVertical, Pencil, Trash } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Alert } from "@/components/ui/alert";
import { permissions } from "@unkey/db/src/schema";
import Loading from "./loading";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

type Props = {
  permissions: Permission[];
  keyId: string;
};

export const Workspace: React.FC<Props> = ({ keyId, permissions }) => {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center">
          <CardTitle className="flex grow shrink-0">Workspace</CardTitle>
          <Dialog>
            <DialogTrigger asChild>
              <Button className="flex grow-0 shrink-0" size="icon">
                <Pencil className="w-4 h-4" />
              </Button>
            </DialogTrigger>

            <DialogContent className="sm:max-w-[425px]">
              <DialogHeader>
                <DialogTitle>Workspace Permissions</DialogTitle>
                <DialogDescription>Manage workspace permissions</DialogDescription>
                {Object.entries(workspacePermissions).map(([category, allPermissions]) => (
                  <div className="flex flex-col gap-2">
                    <span className="font-medium">{category}</span>{" "}
                    <div className="flex flex-col gap-1">
                      {Object.entries(allPermissions).map(
                        ([action, { description, permission }]) => {
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
                        },
                      )}
                    </div>
                  </div>
                ))}
              </DialogHeader>
            </DialogContent>
          </Dialog>
        </div>
        <CardDescription>Manage workspace permissions</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col gap-4">
          {Object.entries(workspacePermissions)
            .filter(([_category, allPermissions]) => {
              return Object.entries(allPermissions).some(
                ([_action, { description: _description, permission }]) => {
                  return permissions.some((p) => p.name === permission);
                },
              );
            })
            .map(([category, allPermissions]) => (
              <div className="flex flex-col gap-2">
                <span className="font-medium">{category}</span>{" "}
                <div className="flex flex-col gap-1">
                  {Object.entries(allPermissions)
                    .filter(([_action, { description: _description, permission }]) => {
                      return permissions.some((p) => p.name === permission);
                    })
                    .map(([action, { description, permission }]) => {
                      return (
                        <div className="flex items-center gap-8">
                          <div className="w-1/3 ">
                            <Tooltip>
                              <TooltipTrigger className="flex items-center gap-2">
                                <Label className="text-xs text-content">{action}</Label>
                              </TooltipTrigger>
                              <TooltipContent className="flex items-center gap-2">
                                <span className="font-mono text-sm font-medium">{permission}</span>
                                <CopyButton value={permission} />
                              </TooltipContent>
                            </Tooltip>
                          </div>

                          <p className="w-2/3 text-xs text-content-subtle">{description}</p>
                        </div>
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
