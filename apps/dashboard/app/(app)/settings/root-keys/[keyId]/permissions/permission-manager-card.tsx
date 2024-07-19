import { CopyButton } from "@/components/dashboard/copy-button";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { Permission } from "@unkey/db";
import { Check, Pencil } from "lucide-react";
import { PermissionToggle } from "./permission_toggle";
import type { UnkeyPermissions } from "./permissions";

type PermissionManagerCardProps = {
  permissions: Permission[];
  keyId: string;

  permissionsStructure: Record<string, UnkeyPermissions>;

  permissionManagerTitle: string;
  permissionManagerDescription: string;

  // TODO: Add the ability to act like an accordion, initially collapsed (prop should be called `expandable`)
};

// TODO: Add a visualization for when there's no active permissions
export function PermissionManagerCard(props: PermissionManagerCardProps) {
  const activePermissions = Object.entries(props.permissionsStructure).filter(
    ([_category, allPermissions]) => {
      const amountActiveRules = Object.entries(allPermissions).filter(
        ([_action, { description: _description, permission }]) => {
          return props.permissions.some((p) => p.name === permission);
        },
      );

      return amountActiveRules.length > 0;

      console.log(props.permissionManagerTitle, b.length);

      // return Object.entries(allPermissions).some(
      //   ([_action, { description: _description, permission }]) => {
      //     return props.permissions.some((p) => {
      //       console.log({ name: p.name, permission });
      //       return p.name === permission;
      //     });
      //   },
      // );
    },
  );
  // const b=  Object.entries(allPermissions)
  // .filter(([_action, { description: _description, permission }]) => {
  //   return props.permissions.some((p) => p.name === permission);
  // })

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center">
          <CardTitle className="flex grow shrink-0">{props.permissionManagerTitle}</CardTitle>
          <Dialog>
            <DialogTrigger asChild>
              <Button className="flex grow-0 shrink-0" size="icon">
                <Pencil className="w-4 h-4" />
              </Button>
            </DialogTrigger>

            <DialogContent className="sm:max-w-[640px] max-h-[70vh] overflow-y-scroll">
              <DialogHeader>
                <DialogTitle>{props.permissionManagerTitle}</DialogTitle>
                <DialogDescription>{props.permissionManagerDescription}</DialogDescription>
              </DialogHeader>

              <div className="flex flex-col w-full gap-4">
                {Object.entries(props.permissionsStructure).map(([category, allPermissions]) => (
                  <div className="flex flex-col gap-2">
                    <span className="font-medium">{category}</span>{" "}
                    <div className="flex flex-col gap-1">
                      {Object.entries(allPermissions).map(
                        ([action, { description, permission }]) => {
                          return (
                            <PermissionToggle
                              key={action}
                              rootKeyId={props.keyId}
                              permissionName={permission}
                              label={action}
                              description={description}
                              checked={props.permissions.some((p) => p.name === permission)}
                            />
                          );
                        },
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </DialogContent>
          </Dialog>
        </div>
        <CardDescription>{props.permissionManagerDescription}</CardDescription>
      </CardHeader>
      <CardContent className="text-sm">
        <div className="flex flex-col gap-4">
          {activePermissions.length === 0 && (
            <p className="text-content-subtle">
              There are no active rules. To get started, edit the permissions.
            </p>
          )}
          {activePermissions.map(([category, allPermissions]) => (
            <div className="flex flex-col gap-2">
              <span className="font-medium">{category}</span>{" "}
              <div className="flex flex-col gap-1">
                {Object.entries(allPermissions)
                  .filter(([_action, { description: _description, permission }]) => {
                    return props.permissions.some((p) => p.name === permission);
                  })
                  .map(([action, { description, permission }]) => {
                    return (
                      <div className="flex items-center gap-8">
                        <div className="w-1/3 ">
                          <Tooltip>
                            <TooltipTrigger className="flex items-center gap-2">
                              <Check className="w-4 h-4 text-success" />
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
}
