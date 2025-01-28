import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Clipboard, InputSearch, PenWriting3 } from "@unkey/icons";
import { type PropsWithChildren, useState } from "react";

export const TableActionPopover = ({ children }: PropsWithChildren) => {
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger onClick={(e) => e.stopPropagation()}>{children}</PopoverTrigger>
      <PopoverContent
        className="w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg"
        align="start"
      >
        <div className="flex flex-col gap-1">
          <PopoverHeader />
          <div
            className="flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer
            hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none"
          >
            <span className="text-[13px] text-accent-12 font-medium">Override</span>
            <PenWriting3 />
          </div>

          <div
            className="flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer
            hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none"
          >
            <span className="text-[13px] text-accent-12 font-medium">Copy Identifier</span>
            <Clipboard />
          </div>

          <div
            className="flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer
            hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none"
          >
            <span className="text-[13px] text-accent-12 font-medium">Search for Identifier</span>
            <InputSearch />
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = () => {
  return (
    <div className="flex w-full justify-between items-center px-2 py-1">
      <span className="text-gray-9 text-[13px]">Actions...</span>
    </div>
  );
};
