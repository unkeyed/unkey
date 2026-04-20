import { cn } from "@/lib/utils";
import { Key2, Page2, Tag } from "@unkey/icons";
import { CopyButton } from "@unkey/ui";
import type { ReactNode } from "react";

type AssignedItemKind = "roles" | "keys" | "slug";

const ICONS: Record<AssignedItemKind, ReactNode> = {
  roles: <Tag iconSize="md-medium" className="opacity-50" />,
  keys: <Key2 iconSize="md-medium" className="opacity-50" />,
  slug: <Page2 iconSize="md-medium" className="opacity-50" />,
};

const LABELS: Record<AssignedItemKind, { singular: string; plural: string }> = {
  roles: { singular: "Role", plural: "Roles" },
  keys: { singular: "Key", plural: "Keys" },
  slug: { singular: "Slug", plural: "Slugs" },
};

export const AssignedItemsCell = ({
  totalCount,
  value,
  isSelected = false,
  kind,
}: {
  totalCount?: number;
  value?: string;
  isSelected?: boolean;
  kind: AssignedItemKind;
}) => {
  const icon = ICONS[kind];
  const getDisplayText = (count: number) => {
    return count === 1 ? LABELS[kind].singular : LABELS[kind].plural;
  };

  const itemClassName = cn(
    "font-mono rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed text-grayA-12",
    isSelected ? "bg-grayA-4 border-grayA-7" : "bg-grayA-3 border-grayA-6 group-hover:bg-grayA-4",
  );

  const emptyClassName = cn(
    "rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed bg-grayA-2",
    isSelected ? "border-grayA-7 text-grayA-9" : "border-grayA-6 text-grayA-8",
  );

  if (kind === "slug") {
    if (!value) {
      return (
        <div className="flex flex-col gap-1 py-2 max-w-[200px] animate-in fade-in slide-in-from-top-2 duration-300">
          <div className={emptyClassName}>
            {icon}
            <span className="text-grayA-9 text-xs">No slug</span>
          </div>
        </div>
      );
    }

    return (
      <div className="flex flex-col gap-1 py-2 max-w-[200px] animate-in fade-in slide-in-from-top-2 duration-300">
        <div
          className={cn(itemClassName, "animate-in fade-in slide-in-from-left-2", "group")}
          style={{ animationDelay: "50ms" }}
        >
          {icon}
          <div className="text-grayA-11 text-xs max-w-[150px] truncate" title={value}>
            {value}
          </div>
          <CopyButton
            value={value}
            variant="ghost"
            className="h-4 w-4 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100 transition-opacity"
          />
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
          {totalCount} {getDisplayText(totalCount)}
        </div>
      </div>
    </div>
  );
};
