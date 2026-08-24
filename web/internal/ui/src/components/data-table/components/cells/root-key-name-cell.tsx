import { cn } from "../../../../lib/utils";

export type RootKeyNameCellProps = {
  name?: string;
  isSelected?: boolean;
};

export const RootKeyNameCell = ({ name }: RootKeyNameCellProps) => {
  return (
    <div className="flex flex-col items-start px-[18px] py-[6px]">
      <div className="flex w-full items-center">
        <div
          className={cn(
            "min-w-0 flex-1 truncate font-medium leading-4 text-[13px]",
            name ? "text-accent-12" : "text-gray-9 italic font-normal",
          )}
        >
          {name ?? "Unnamed root key"}
        </div>
      </div>
    </div>
  );
};
