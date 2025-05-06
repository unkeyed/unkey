import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Dots } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { type FC, type PropsWithChildren, useEffect, useRef, useState } from "react";

export type ActionComponentProps = {
  isOpen: boolean;
  onClose: () => void;
};

export type MenuItem = {
  id: string;
  label: string;
  icon: React.ReactNode;
  onClick?: (e: React.MouseEvent<Element, MouseEvent> | React.KeyboardEvent<Element>) => void;
  className?: string;
  disabled?: boolean;
  divider?: boolean;
  ActionComponent?: FC<ActionComponentProps>;
};

type BaseTableActionPopoverProps = PropsWithChildren<{
  items: MenuItem[];
  align?: "start" | "end";
}>;

export const KeysTableActionPopover = ({ items, align = "end" }: BaseTableActionPopoverProps) => {
  const [enabledItem, setEnabledItem] = useState<string>();
  const [open, setOpen] = useState(false);
  const [focusIndex, setFocusIndex] = useState(0);
  const menuItems = useRef<HTMLDivElement[]>([]);

  useEffect(() => {
    if (open) {
      const firstEnabledIndex = items.findIndex((item) => !item.disabled);
      setFocusIndex(firstEnabledIndex >= 0 ? firstEnabledIndex : 0);
      if (firstEnabledIndex >= 0) {
        menuItems.current[firstEnabledIndex]?.focus();
      }
    }
  }, [open, items]);

  const handleActionSelection = (value: string) => {
    setEnabledItem(value);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger onClick={(e) => e.stopPropagation()}>
        <button
          type="button"
          className={cn(
            "group-data-[state=open]:bg-gray-6 group-hover:bg-gray-6 group size-5 p-0 rounded m-0 items-center flex justify-center",
            "border border-gray-6 hover:border-gray-8 ring-2 ring-transparent focus-visible:ring-gray-7 focus-visible:border-gray-7",
          )}
        >
          <Dots className="group-hover:text-gray-12 text-gray-11" size="sm-regular" />
        </button>
      </PopoverTrigger>
      <PopoverContent
        className="w-60 bg-gray-1 dark:bg-black drop-shadow-2xl border-gray-6 rounded-lg p-0"
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
        {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
        <div role="menu" onClick={(e) => e.stopPropagation()} className="py-2">
          {items.map((item, index) => (
            <div key={item.id}>
              <div className="px-2">
                {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
                <div
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
                      item.onClick?.(e);

                      if (!item.ActionComponent) {
                        setOpen(false);
                      }

                      setEnabledItem(item.id);
                    }
                  }}
                >
                  <div className="text-gray-9 group-hover:text-gray-12 group-focus:text-gray-12">
                    {item.icon}
                  </div>
                  <span className="text-[13px] font-medium">{item.label}</span>
                </div>
              </div>
              {item.divider && <div className="h-[1px] bg-grayA-3 w-full my-2" />}
              {item.ActionComponent && enabledItem === item.id && (
                <item.ActionComponent isOpen onClose={() => handleActionSelection("none")} />
              )}
            </div>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};
