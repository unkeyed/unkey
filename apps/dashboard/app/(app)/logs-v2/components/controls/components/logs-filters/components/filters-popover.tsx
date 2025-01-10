import { useKeyboardShortcut } from "@/app/(app)/logs-v2/hooks/use-keyboard-shortcut";
import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { CaretRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { type PropsWithChildren, useState } from "react";
import { MethodsFilter } from "./methods-filter";
import { PathsFilter } from "./paths-filter";
import { StatusFilter } from "./status-filter";

type FilterItemConfig = {
  id: string;
  label: string;
  shortcut?: string;
  component: React.ReactNode;
};

const FILTER_ITEMS: FilterItemConfig[] = [
  {
    id: "status",
    label: "Status",
    shortcut: "s",
    component: <StatusFilter />,
  },
  {
    id: "method",
    label: "Method",
    shortcut: "m",
    component: <MethodsFilter />,
  },
  {
    id: "path",
    label: "Path",
    shortcut: "p",
    component: <PathsFilter />,
  },
];

export const FiltersPopover = ({ children }: PropsWithChildren) => {
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
        <div className="flex flex-col gap-2">
          <PopoverHeader />
          {FILTER_ITEMS.map((item) => (
            <FilterItem key={item.id} {...item} />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = () => {
  return (
    <div className="flex w-full justify-between items-center px-2 py-1">
      <span className="text-gray-9 text-[13px]">Filters...</span>
      <KeyboardButton shortcut="F" />
    </div>
  );
};

export const FilterItem = ({ label, shortcut, component }: FilterItemConfig) => {
  const [open, setOpen] = useState(false);

  // Add keyboard shortcut for each filter item when main filter is open
  useKeyboardShortcut(
    { key: shortcut || "", meta: true },
    () => {
      setOpen(true);
    },
    { preventDefault: true },
  );

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div className="flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer hover:bg-gray-3 data-[state=open]:bg-gray-3">
          <div
            className="flex gap-2 items-center"
            aria-label={label}
            aria-haspopup="true"
            title={`Press '⌘${shortcut?.toUpperCase()}' to toggle ${label} filters`}
          >
            {shortcut && <KeyboardButton shortcut={shortcut} modifierKey="⌘" />}
            <span className="text-[13px] text-accent-12 font-medium">{label}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Button variant="ghost" size="icon" tabIndex={-1} className="size-5 [&_svg]:size-2">
              <CaretRight className="text-gray-7 group-hover:text-gray-10" />
            </Button>
          </div>
        </div>
      </PopoverTrigger>
      <PopoverContent
        className="w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-0 border-gray-6 rounded-lg"
        side="right"
        align="start"
        sideOffset={12}
      >
        {component}
      </PopoverContent>
    </Popover>
  );
};
