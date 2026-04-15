import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { Page2 } from "@unkey/icons";
import { Checkbox } from "@unkey/ui";
import { cn } from "@/lib/utils";

type PermissionNameCellProps = {
  permission: Permission;
  isChecked: boolean;
  isHovered: boolean;
  onToggleSelection: (permissionId: string) => void;
  onHover: (name: string | null) => void;
};

export const PermissionNameCell = ({
  permission,
  isChecked,
  isHovered,
  onToggleSelection,
  onHover,
}: PermissionNameCellProps) => {
  const iconContainer = (
    <div
      className={cn(
        "size-5 rounded-sm flex items-center justify-center border border-grayA-3 transition-all duration-100",
        "bg-grayA-3",
        isChecked && "bg-grayA-5",
      )}
      onMouseEnter={() => onHover(permission.name)}
      onMouseLeave={() => onHover(null)}
    >
      {!isChecked && !isHovered && (
        <Page2 iconSize="sm-regular" className="text-gray-12 cursor-pointer" />
      )}
      {(isChecked || isHovered) && (
        <Checkbox
          checked={isChecked}
          className="size-4 [&_svg]:size-3"
          onClick={(e) => e.stopPropagation()}
          onCheckedChange={() => onToggleSelection(permission.permissionId)}
        />
      )}
    </div>
  );

  return (
    <div className="flex flex-col items-start px-[18px] py-[6px]">
      <div className="flex gap-4 items-center">
        {iconContainer}
        <div className="flex flex-col gap-1 text-xs">
          <div className="font-medium truncate text-accent-12 leading-4 text-[13px] max-w-[120px]">
            {permission.name}
          </div>
          {permission.description ? (
            <span
              className="font-sans text-accent-9 truncate max-w-[180px] text-xs"
              title={permission.description}
            >
              {permission.description}
            </span>
          ) : (
            <span className="font-sans text-accent-9 truncate max-w-[180px] text-xs italic">
              No description
            </span>
          )}
        </div>
      </div>
    </div>
  );
};
