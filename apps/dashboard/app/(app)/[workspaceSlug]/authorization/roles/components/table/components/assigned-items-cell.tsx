import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Key2, Page2 } from "@unkey/icons";

export const AssignedItemsCell = ({
  roleId,
  kind,
  isSelected = false,
}: {
  roleId: string;
  kind: "keys" | "permissions";
  isSelected?: boolean;
}) => {
  const { data: keysData, isLoading: keysLoading } =
    trpc.authorization.roles.connectedKeys.useQuery(
      { roleId },
      {
        enabled: kind === "keys",
        staleTime: 5 * 60 * 1000,
      },
    );
  const { data: permissionsData, isLoading: permissionsLoading } =
    trpc.authorization.roles.connectedPerms.useQuery(
      { roleId },
      {
        enabled: kind === "permissions",
        staleTime: 5 * 60 * 1000,
      },
    );

  const data = kind === "keys" ? keysData : permissionsData;
  const isLoading = kind === "keys" ? keysLoading : permissionsLoading;
  const totalCount = data?.totalCount;

  const icon =
    kind === "keys" ? (
      <Key2 iconSize="md-medium" className="opacity-50" />
    ) : (
      <Page2 iconSize="md-medium" className="opacity-50" />
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
          {icon}
          <div className="h-2 w-20 bg-grayA-3 rounded animate-pulse" />
        </div>
      </div>
    );
  }

  if (!totalCount) {
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
      <div
        className={cn(itemClassName, "animate-in fade-in slide-in-from-left-2")}
        style={{ animationDelay: "50ms" }}
      >
        {icon}
        <div className="text-grayA-11 text-xs max-w-[150px] truncate">
          {totalCount} {getDisplayText(totalCount, kind)}
        </div>
      </div>
    </div>
  );
};

const getDisplayText = (count: number, kind: "keys" | "permissions") => {
  if (count === 1) {
    return kind === "keys" ? "Key" : "Permission";
  }
  return kind === "keys" ? "Keys" : "Permissions";
};
