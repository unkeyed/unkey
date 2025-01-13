import { useKeyboardShortcut } from "@/app/(app)/logs-v2/hooks/use-keyboard-shortcut";
import { KeyboardButton } from "@/components/keyboard-button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { type PropsWithChildren, useState } from "react";

type SearchItemConfig = {
  label: string;
  shortcut?: string;
};

const SEARCH_ITEMS: SearchItemConfig[] = [
  {
    label: "requestId",
    shortcut: "r",
  },
  {
    label: "host",
    shortcut: "h",
  },
  {
    label: "method",
    shortcut: "m",
  },

  {
    label: "path",
    shortcut: "p",
  },
];

export const SearchPopover = ({ children }: PropsWithChildren) => {
  const [open, setOpen] = useState(false);

  useKeyboardShortcut("f", () => {
    setOpen((prev) => !prev);
  });

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent
        className="w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg"
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

export const SearchItem = ({ label, shortcut }: SearchItemConfig) => {
  // Add keyboard shortcut for each filter item when main filter is open
  useKeyboardShortcut({ key: shortcut || "", meta: true }, () => {}, {
    preventDefault: true,
  });

  return (
    <div className="flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer group hover:bg-gray-3 data-[state=open]:bg-gray-3">
      <span className="text-xs text-accent-11 font-mono bg-gray-3 px-2 py-0.5 rounded-md group-hover:bg-gray-2">
        {label}
      </span>
      {shortcut && (
        <KeyboardButton
          shortcut={shortcut}
          modifierKey="⌘"
          role="presentation"
          aria-haspopup="true"
          title={`Press '⌘${shortcut?.toUpperCase()}' to toggle ${label} options`}
        />
      )}
    </div>
  );
};
