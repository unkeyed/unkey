"use client";

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Permission } from "@unkey/db";
import { type PropsWithChildren, useMemo, useState } from "react";
import { PermissionToggle } from "./permission_toggle";
import { apiPermissions } from "./permissions";

export function DialogAddPermissionsForAPI(
  props: PropsWithChildren<{
    keyId: string;
    apis: { id: string; name: string }[];
    permissions: Permission[];
  }>,
) {
  const apisWithoutPermission = props.apis.filter((api) => {
    const apiPermissionsStructure = apiPermissions(api.id);
    const hasActivePermissions = Object.entries(apiPermissionsStructure).some(
      ([_category, allPermissions]) => {
        const amountActiveRules = Object.entries(allPermissions).filter(
          ([_action, { description: _description, permission }]) => {
            return props.permissions.some((p) => p.name === permission);
          },
        );

        return amountActiveRules.length > 0;
      },
    );

    return !hasActivePermissions;
  });

  const [selectedApiId, setSelectedApiId] = useState<string>("");
  const selectedApi = useMemo(
    () => props.apis.find((api) => api.id === selectedApiId),
    [selectedApiId],
  );

  const isSelectionDisabled =
    selectedApi && !apisWithoutPermission.some((api) => api.id === selectedApi.id);

  const options = apisWithoutPermission.reduce(
    (map, api) => {
      map[api.id] = api.name;
      return map;
    },
    {} as Record<string, string>,
  );

  function onOpenChange() {
    setSelectedApiId("");
  }

  return (
    <Dialog onOpenChange={onOpenChange}>
      {/* Trigger should be in here */}
      {props.children}

      <DialogContent className="sm:max-w-[640px] max-h-[70vh] overflow-y-scroll">
        <DialogHeader>
          <DialogTitle>Setup permissions for an API</DialogTitle>
          <Select
            value={selectedApiId}
            onValueChange={setSelectedApiId}
            disabled={isSelectionDisabled}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select an API" />
            </SelectTrigger>
            <SelectContent>
              {Object.entries(options).map(([id, label]) => (
                <SelectItem key={id} value={id}>
                  {label}
                </SelectItem>
              ))}
              {selectedApi && !Object.entries(options).some(([id]) => id === selectedApiId) && (
                <SelectItem value={selectedApiId}>{selectedApi.name}</SelectItem>
              )}
            </SelectContent>
          </Select>
        </DialogHeader>

        {selectedApiId !== "" && (
          <div className="flex flex-col w-full gap-4">
            {Object.entries(apiPermissions(selectedApiId)).map(([category, allPermissions]) => (
              <div className="flex flex-col gap-2">
                <span className="font-medium">{category}</span>{" "}
                <div className="flex flex-col gap-1">
                  {Object.entries(allPermissions).map(([action, { description, permission }]) => {
                    return (
                      <PermissionToggle
                        key={action}
                        rootKeyId={props.keyId}
                        permissionName={permission}
                        label={action}
                        description={description}
                        checked={props.permissions.some((p) => p.name === permission)}
                        preventDisabling={!selectedApi}
                        preventEnabling={!selectedApi}
                      />
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
