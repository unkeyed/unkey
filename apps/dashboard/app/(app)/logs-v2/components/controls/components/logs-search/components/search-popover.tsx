import { KeyboardButton } from "@/components/keyboard-button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { type PropsWithChildren, useState } from "react";

type SearchItemConfig = {
  label: string;
  description: string;
};

const SEARCH_ITEMS: SearchItemConfig[] = [
  {
    label: "requestId",
    description: "Unique request ID",
  },
  {
    label: "host",
    description: "Server hostname",
  },
  {
    label: "method",
    description: "HTTP method",
  },
  {
    label: "path",
    description: "Request URL path",
  },
];

export const SearchPopover = ({ children }: PropsWithChildren) => {
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent
        onOpenAutoFocus={(e) => e.preventDefault()}
        className="w-80 bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg"
        align="start"
      >
        <div className="flex flex-col gap-1">
          <PopoverHeader />
          {SEARCH_ITEMS.map((item) => (
            <SearchItem key={item.label} {...item} />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = () => {
  return (
    <div className="flex w-full justify-between items-center px-2 py-1">
      <span className="text-gray-9 text-[13px]">
        Search and filter with AI...
      </span>
      <KeyboardButton shortcut="S" />
    </div>
  );
};

export const SearchItem = ({ label, description }: SearchItemConfig) => {
  return (
    <div className="flex w-full items-center px-2 py-1.5 rounded-lg group cursor-pointer group font-mono text-xs gap-2 group">
      <span className="text-accent-11 bg-gray-3 px-2 py-0.5 rounded-md ">
        {label}
      </span>
      <span className="text-accent-9">{description}</span>
    </div>
  );
};
