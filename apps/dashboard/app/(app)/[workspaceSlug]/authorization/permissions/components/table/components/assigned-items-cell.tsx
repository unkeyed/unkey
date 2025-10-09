import { cn } from "@/lib/utils";
import { Key2, Page2, Tag } from "@unkey/icons";

export const AssignedItemsCell = ({
  totalCount,
  value,
  isSelected = false,
  kind,
}: {
  totalCount?: number;
  value?: string;
  isSelected?: boolean;
  kind: "roles" | "keys" | "permissions" | "slug";
}) => {
  const getIcon = () => {
    switch (kind) {
      case "roles":
        return <Tag iconSize="md-medium" className="opacity-50" />;
      case "keys":
        return <Key2 iconSize="md-medium" className="opacity-50" />;
      case "slug":
        return <Page2 iconSize="md-medium" className="opacity-50" />;
      default:
        throw new Error(`Invalid type: ${kind}`);
    }
  };

  const getDisplayText = (count: number) => {
    if (count === 1) {
      switch (kind) {
        case "roles":
          return "Role";
        case "keys":
          return "Key";
        case "permissions":
          return "Permission";
        default:
          throw new Error(`Invalid type: ${kind}`);
      }
    }

    switch (kind) {
      case "roles":
        return "Roles";
      case "keys":
        return "Keys";
      case "permissions":
        return "Permissions";
      default:
        throw new Error(`Invalid type: ${kind}`);
    }
  };

  const itemClassName = cn(
    "font-mono rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed text-grayA-12",
    isSelected
      ? "bg-grayA-4 border-grayA-7"
      : "bg-grayA-3 border-grayA-6 group-hover:bg-grayA-4"
  );

  const emptyClassName = cn(
    "rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed bg-grayA-2",
    isSelected ? "border-grayA-7 text-grayA-9" : "border-grayA-6 text-grayA-8"
  );

  if (kind === "slug") {
    if (!value) {
      return (
        <div className="flex flex-col gap-1 py-2 max-w-[200px] animate-in fade-in slide-in-from-top-2 duration-300">
          <div className={emptyClassName}>
            {getIcon()}
            <span className="text-grayA-9 text-xs">No slug</span>
          </div>
        </div>
      );
    }

    return (
      <div className="flex flex-col gap-1 py-2 max-w-[200px] animate-in fade-in slide-in-from-top-2 duration-300">
        <div
          className={cn(
            itemClassName,
            "animate-in fade-in slide-in-from-left-2"
          )}
          style={{ animationDelay: "50ms" }}
        >
          {getIcon()}
          <div
            className="text-grayA-11 text-xs max-w-[150px] truncate"
            title={value}
          >
            {value}
          </div>
        </div>
      </div>
    );
  }

  if (!totalCount) {
    return (
      <div className="flex flex-col gap-1 py-2 max-w-[200px] animate-in fade-in slide-in-from-top-2 duration-300">
        <div className={emptyClassName}>
          {getIcon()}
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
        {getIcon()}
        <div className="text-grayA-11 text-xs max-w-[150px] truncate">
          {totalCount} {getDisplayText(totalCount)}
        </div>
      </div>
    </div>
  );
};
