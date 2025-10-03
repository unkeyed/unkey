import type { IconProps } from "@unkey/icons/src/props";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

type FilterButtonProps = {
  isActive: boolean;
  count: number | string;
  onClick: () => void;
  icon: React.ComponentType<IconProps>;
  label: string;
};

export const FilterButton = ({
  isActive,
  count,
  onClick,
  icon: Icon,
  label,
}: FilterButtonProps) => (
  <Button
    variant="primary"
    className={cn(
      "text-xs h-[26px] border-none hover:bg-grayA-4",
      isActive ? "bg-gray-12 hover:bg-grayA-12" : "bg-grayA-3 text-grayA-9",
    )}
    onClick={onClick}
  >
    <Icon size="sm-regular" className={isActive ? "" : "text-grayA-9 !size-3"} />
    <span className={isActive ? "" : "text-grayA-9"}>{label}</span>
    <div className="rounded w-[22px] h-[18px] flex items-center justify-center text-[10px] leading-4 bg-gray-6 text-black dark:text-white">
      {count}
    </div>
  </Button>
);
