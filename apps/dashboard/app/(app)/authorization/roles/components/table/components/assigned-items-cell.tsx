import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { HandHoldingKey, Key2 } from "@unkey/icons";
import React from "react";

export const AssignedItemsCell = ({
  roleId,
  type,
  isSelected = false,
}: {
  roleId: string;
  type: "keys" | "permissions";
  isSelected?: boolean;
}) => {
  const { data: keysData, isLoading: keysLoading } =
    trpc.authorization.roles.connectedKeys.useQuery(
      { roleId, limit: 3 },
      {
        enabled: type === "keys",
        staleTime: 5 * 60 * 1000,
      },
    );

  const { data: permissionsData, isLoading: permissionsLoading } =
    trpc.authorization.roles.connectedPerms.useQuery(
      { roleId, limit: 3 },
      {
        enabled: type === "permissions",
        staleTime: 5 * 60 * 1000,
      },
    );

  const data = type === "keys" ? keysData : permissionsData;
  const isLoading = type === "keys" ? keysLoading : permissionsLoading;
  const items = data?.items || [];
  const totalCount = data?.totalCount;
  const hasMore = totalCount && totalCount > items.length;

  const icon =
    type === "keys" ? (
      <Key2 size="md-regular" />
    ) : (
      <HandHoldingKey className="size-3" size="md-regular" />
    );

  const itemClassName = cn(
    "font-mono rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed text-grayA-12",
    isSelected ? "bg-grayA-4 border-grayA-7" : "bg-grayA-3 border-grayA-6 group-hover:bg-grayA-4",
  );

  const emptyClassName = cn(
    "rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed bg-grayA-2",
    isSelected ? "border-grayA-7 text-grayA-9" : "border-grayA-6 text-grayA-8",
  );

  if (isLoading) {
    return (
      <div className="flex flex-col gap-1 py-2 max-w-[200px] animate-in fade-in duration-300">
        <div className="rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 border border-dashed bg-grayA-3 border-grayA-6 animate-pulse h-[22px]">
          {React.cloneElement(icon, { className: "opacity-50" })}
          <div className="h-2 w-16 bg-grayA-3 rounded animate-pulse" />
        </div>
        <div className="rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 border border-dashed bg-grayA-3 border-grayA-6 animate-pulse h-[22px]">
          {React.cloneElement(icon, { className: "opacity-50" })}
          <div className="h-2 w-12 bg-grayA-3 rounded animate-pulse" />
        </div>
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="flex flex-col gap-1 py-2 max-w-[200px] animate-in fade-in slide-in-from-top-2 duration-300">
        <div className={emptyClassName}>
          {icon}
          <span className="text-grayA-9 text-xs">None assigned</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 py-2 max-w-[200px] animate-in fade-in slide-in-from-top-2 duration-300">
      {items.map((item, index) => (
        <div
          className={cn(itemClassName, "animate-in fade-in slide-in-from-left-2")}
          key={`${item}-${
            // biome-ignore lint/suspicious/noArrayIndexKey: Since item names are not unique sometimes we get overlapping names
            index
          }`}
          style={{ animationDelay: `${index * 50}ms` }}
        >
          {icon}
          <span className="text-grayA-11 text-xs max-w-[150px] truncate">{item}</span>
        </div>
      ))}
      {hasMore && (
        <div
          className={cn(itemClassName, "animate-in fade-in slide-in-from-left-2")}
          style={{ animationDelay: `${items.length * 50}ms` }}
        >
          <span className="text-grayA-9 text-xs max-w-[150px] truncate">
            {totalCount - items.length} more {type}...
          </span>
        </div>
      )}
    </div>
  );
};
