import { useKeyboardShortcut } from "@/app/(app)/logs-v2/hooks/use-keyboard-shortcut";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { type PropsWithChildren, useState } from "react";

export const LogsFilters = () => {
  return (
    <FiltersPopover>
      <div className="group">
        <Button variant="ghost" className="group-data-[state=open]:bg-accent-4">
          <BarsFilter className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">Filter</span>
        </Button>
      </div>
    </FiltersPopover>
  );
};

function FiltersPopover({ children }: PropsWithChildren) {
  const [open, setOpen] = useState(false);

  useKeyboardShortcut("f", () => {
    setOpen((prev) => !prev);
  });

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent className="w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-2" align="start">
        <div className="flex flex-col">
          <div className="flex w-full justify-between items-center px-[6px] py-[2px]">
            <span className="text-gray-9 text-[13px]">Filters...</span>
            <Button
              variant="ghost"
              size="icon"
              tabIndex={-1}
              className="text-xs size-5 rounded-[5px] bg-gray-3 text-gray-9 border-gray-8 border border-solid"
            >
              F
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
