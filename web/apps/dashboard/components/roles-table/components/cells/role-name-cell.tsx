import { cn } from "@/lib/utils";
import { Tag } from "@unkey/icons";
import { Checkbox } from "@unkey/ui";

type RoleNameCellProps = {
  name: string;
  description?: string;
  isSelected: boolean;
  isHovered: boolean;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
  onCheckedChange: () => void;
};

export const RoleNameCell = ({
  name,
  description,
  isSelected,
  isHovered,
  onMouseEnter,
  onMouseLeave,
  onCheckedChange,
}: RoleNameCellProps) => {
  return (
    <div className="flex flex-col items-start px-[18px] py-[6px]">
      <div className="flex gap-4 items-center">
        <div
          className={cn(
            "size-5 rounded-sm flex items-center justify-center border border-grayA-3 transition-all duration-100",
            "bg-grayA-3",
            isSelected && "bg-grayA-5",
          )}
          onMouseEnter={onMouseEnter}
          onMouseLeave={onMouseLeave}
        >
          {!isSelected && !isHovered && (
            <Tag iconSize="sm-regular" className="text-gray-12 cursor-pointer w-[12px] h-[12px]" />
          )}
          {(isSelected || isHovered) && (
            <Checkbox
              checked={isSelected}
              className="size-4 [&_svg]:size-3"
              onClick={(e) => e.stopPropagation()}
              onCheckedChange={onCheckedChange}
            />
          )}
        </div>
        <div className="flex flex-col gap-1 text-xs">
          <div className="font-medium truncate text-accent-12 leading-4 text-[13px] max-w-[120px]">
            {name}
          </div>
          {description ? (
            <span
              className="font-sans text-accent-9 truncate max-w-[180px] text-xs"
              title={description}
            >
              {description}
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
