import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@unkey/ui/src/lib/utils";
import { type PropsWithChildren, useEffect, useRef, useState } from "react";
import { TableActionButton } from "./table-action-button";

export type MenuItem = {
  id: string;
  label: string;
  icon: React.ReactNode;
  onClick: (e: React.MouseEvent<Element, MouseEvent> | React.KeyboardEvent<Element>) => void;
  className?: string;
  disabled?: boolean;
};

type BaseTableActionPopoverProps = PropsWithChildren<{
  items: MenuItem[];
  align?: "start" | "end";
  headerContent?: React.ReactNode;
}>;

export const TableActionPopover = ({
  items,
  align = "end",
  headerContent,
  children,
}: BaseTableActionPopoverProps) => {
  const [open, setOpen] = useState(false);
  const [focusIndex, setFocusIndex] = useState(0);
  const menuItems = useRef<HTMLDivElement[]>([]);

  useEffect(() => {
    if (open) {
      // Find first non-disabled item
      const firstEnabledIndex = items.findIndex((item) => !item.disabled);
      setFocusIndex(firstEnabledIndex >= 0 ? firstEnabledIndex : 0);
      if (firstEnabledIndex >= 0) {
        menuItems.current[firstEnabledIndex]?.focus();
      }
    }
  }, [open, items]);

  const handleKeyDown = (e: React.KeyboardEvent<Element>) => {
    e.stopPropagation();

    const activeElement = document.activeElement;
    const currentIndex = menuItems.current.findIndex((item) => item === activeElement);
    const itemCount = items.length;

    const findNextEnabledIndex = (startIndex: number, direction: 1 | -1) => {
      let index = startIndex;
      for (let i = 0; i < itemCount; i++) {
        index = (index + direction + itemCount) % itemCount;
        if (!items[index].disabled) {
          return index;
        }
      }
      return startIndex;
    };

    switch (e.key) {
      case "Tab": {
        e.preventDefault();
        const nextIndex = findNextEnabledIndex(currentIndex, e.shiftKey ? -1 : 1);
        setFocusIndex(nextIndex);
        menuItems.current[nextIndex]?.focus();
        break;
      }

      case "j":
      case "ArrowDown": {
        e.preventDefault();
        const nextDownIndex = findNextEnabledIndex(currentIndex, 1);
        setFocusIndex(nextDownIndex);
        menuItems.current[nextDownIndex]?.focus();
        break;
      }

      case "k":
      case "ArrowUp": {
        e.preventDefault();
        const nextUpIndex = findNextEnabledIndex(currentIndex, -1);
        setFocusIndex(nextUpIndex);
        menuItems.current[nextUpIndex]?.focus();
        break;
      }

      case "Escape":
        e.preventDefault();
        setOpen(false);
        break;

      case "Enter":
      case "ArrowRight":
      case "l":
      case " ":
        e.preventDefault();
        if (activeElement === menuItems.current[currentIndex] && !items[currentIndex].disabled) {
          items[currentIndex].onClick(e);
        }
        break;
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger onClick={(e) => e.stopPropagation()}>
        {children ? children : <TableActionButton />}
      </PopoverTrigger>

      <PopoverContent
        className="w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg"
        align={align}
        onOpenAutoFocus={(e) => {
          e.preventDefault();
          const firstEnabledIndex = items.findIndex((item) => !item.disabled);
          if (firstEnabledIndex >= 0) {
            menuItems.current[firstEnabledIndex]?.focus();
          }
        }}
        onCloseAutoFocus={(e) => e.preventDefault()}
        onEscapeKeyDown={(e) => {
          e.preventDefault();
          setOpen(false);
        }}
        onInteractOutside={(e) => {
          e.preventDefault();
          setOpen(false);
        }}
      >
        <div
          className="flex flex-col gap-2"
          role="menu"
          onClick={(e) => e.stopPropagation()}
          onKeyDown={handleKeyDown}
        >
          {headerContent ?? <PopoverHeader />}

          {items.map((item, index) => (
            // biome-ignore lint/a11y/useKeyWithClickEvents: <explanation>
            <div
              key={item.id}
              ref={(el) => {
                if (el) {
                  menuItems.current[index] = el;
                }
              }}
              role="menuitem"
              aria-disabled={item.disabled}
              tabIndex={!item.disabled && focusIndex === index ? 0 : -1}
              className={cn(
                "flex w-full items-center px-2 py-1.5 gap-3 rounded-lg group",
                !item.disabled &&
                  "cursor-pointer hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none focus:bg-gray-3",
                item.disabled && "cursor-not-allowed opacity-50",
                item.className,
              )}
              onClick={(e) => {
                if (!item.disabled) {
                  item.onClick(e);
                  setOpen(false);
                }
              }}
            >
              {item.icon}
              <span className="text-[13px] font-medium">{item.label}</span>
            </div>
          ))}
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
