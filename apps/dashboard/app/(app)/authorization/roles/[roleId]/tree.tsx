"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Switch } from "@/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ChevronRight } from "@unkey/icons";
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
  const entries = Object.entries(nestedPermissions);

  return (
    <div className="flex justify-between w-full">
      <div className="flex justify-between items-start flex-row-reverse bg-gray-2 border border-border rounded-lg p-3 shadow-sm w-full">
        <div className="flex items-center justify-between mb-3 pb-2">
          <div />
          <div className="flex items-center gap-2">
            <div className="text-sm text-accent-12">{openAll ? "Collapse" : "Expand"} All</div>
            <Switch checked={openAll} onCheckedChange={setOpenAll} />
          </div>
        </div>

        <div className="flex flex-col gap-1">
          {entries.map(([k, p]) => (
            <RecursivePermission
              key={p.id}
              k={k}
              {...p}
              roleId={role.id}
              openAll={openAll}
              depth={0}
            />
          ))}
        </div>
      </div>
    </div>
  );
};

export const RecursivePermission: React.FC<
  NestedPermission & {
    k: string;
    roleId: string;
    openAll: boolean;
    depth: number;
  }
> = ({ k, openAll, id, name, permissions, roleId, checked, description, depth }) => {
  const [open, setOpen] = useState(openAll);
  const [hover, setHover] = useState(false);

  useEffect(() => {
    setOpen(openAll);
  }, [openAll]);

  const children = Object.values(permissions);
  const hasChildren = children.length > 0;

  const getBgColor = (isHovering: boolean) => {
    if (!isHovering) {
      return "";
    }
    return "bg-grayA-3";
  };

  if (!hasChildren) {
    return (
      <Tooltip delayDuration={50}>
        <TooltipTrigger
          className={`flex flex-col items-start px-2 first:pt-0  py-1.5 mx-1 rounded transition-colors ${getBgColor(
            hover,
          )}`}
          onMouseEnter={() => setHover(true)}
          onMouseLeave={() => setHover(false)}
        >
          <div className="flex items-center gap-2 w-full">
            <PermissionToggle permissionId={id} roleId={roleId} checked={checked} />
            <pre className="text-sm text-accent-12 font-medium">{k}</pre>
          </div>
          {description && <p className="text-xs text-accent-11 mt-0.5">{description}</p>}
        </TooltipTrigger>
        <TooltipContent
          side="top"
          align="start"
          avoidCollisions={true}
          className="bg-gray-12 text-gray-1 px-3 py-2 border border-accent-6 shadow-md font-medium text-xs"
        >
          <div className="flex items-center justify-start max-w-sm gap-2">
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
    <div className="w-full">
      <div
        className={`flex items-center justify-between px-2 mx-1 rounded transition-colors ${getBgColor(
          hover,
        )}`}
        onMouseEnter={() => setHover(true)}
        onMouseLeave={() => setHover(false)}
      >
        {/* biome-ignore lint/a11y/useKeyWithClickEvents: Simplified click handler */}
        <div
          className="flex items-center gap-2 cursor-pointer text-accent-12 hover:text-accent-11 transition-colors w-full"
          onClick={() => setOpen(!open)}
        >
          <ChevronRight
            className={`size-3 transition-transform duration-200 ${open ? "rotate-90" : ""}`}
          />

          <pre className="text-sm font-medium">{k}</pre>

          <div className="bg-gray-7 rounded h-4 px-1 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
            {children.length}
          </div>
        </div>
      </div>

      {open && (
        <div
          className={`flex flex-col gap-0.5 ml-2 pl-4 border-l border-grayA-5 mt-1 transition-all duration-200 ease-in-out ${
            open ? "opacity-100" : "opacity-0 h-0"
          }`}
        >
          {Object.entries(permissions).map(([k2, p]) => (
            <RecursivePermission
              key={p.id}
              k={k2}
              {...p}
              roleId={roleId}
              openAll={openAll}
              depth={depth + 1}
            />
          ))}
        </div>
      )}
    </div>
  );
};
