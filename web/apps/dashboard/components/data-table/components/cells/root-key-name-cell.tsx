import { cn } from "@/lib/utils";
import { Key2 } from "@unkey/icons";

type RootKeyNameCellProps = {
  name?: string;
  isSelected?: boolean;
};

export const RootKeyNameCell = ({ name, isSelected = false }: RootKeyNameCellProps) => {
  return (
    <div className="flex flex-col items-start px-[18px] py-[6px]">
      <div className="flex gap-3 items-center w-full">
        <div
          className={cn(
            "size-5 rounded flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100",
            "bg-grayA-3",
            isSelected && "bg-grayA-5",
          )}
        >
          <Key2 iconSize="sm-regular" className="text-gray-12" />
        </div>
        <div className="w-[150px]">
          <div
            className={cn(
              "font-medium truncate leading-4 text-[13px]",
              name ? "text-accent-12" : "text-gray-9 italic font-normal",
            )}
          >
            {name ?? "Unnamed Root Key"}
          </div>
        </div>
      </div>
    </div>
  );
};
