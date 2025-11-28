import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Dots } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { type FC, type PropsWithChildren, forwardRef, useEffect, useRef, useState } from "react";

export type ActionComponentProps = {
  isOpen: boolean;
  onClose: () => void;
};

interface ReactLoadableProps {
  isLoading?: boolean;
  pastDelay?: boolean;
  timedOut?: boolean;
  retry?: () => void;
  error?: Error | null;
}

export type MenuItem = {
  id: string;
  label: string;
  icon: React.ReactNode;
  onClick?: (e: React.MouseEvent<Element, MouseEvent> | React.KeyboardEvent<Element>) => void;
  className?: string;
  disabled?: boolean;
  divider?: boolean;
  ActionComponent?: FC<ActionComponentProps>;
  prefetch?: () => Promise<void>;
};

type BaseTableActionPopoverProps = PropsWithChildren<{
  items: MenuItem[];
  align?: "start" | "end";
}>;

export const TableActionPopover = ({
  items,
  align = "end",
  children,
}: BaseTableActionPopoverProps) => {
  const [enabledItem, setEnabledItem] = useState<string>();
  const [open, setOpen] = useState(false);
  const [focusIndex, setFocusIndex] = useState(0);
  const [prefetchedItems, setPrefetchedItems] = useState<Set<string>>(new Set());
  const menuItems = useRef<HTMLDivElement[]>([]);

  useEffect(() => {
    if (open) {
      // Prefetch all items that need prefetching and haven't been prefetched yet
      items
        .filter((item) => item.prefetch && !prefetchedItems.has(item.id))
        .forEach(async (item) => {
          try {
            await item.prefetch?.();
            setPrefetchedItems((prev) => new Set(prev).add(item.id));
          } catch (error) {
            console.error(`Failed to prefetch data for ${item.id}:`, error);
          }
        });

      const firstEnabledIndex = items.findIndex((item) => !item.disabled);
      setFocusIndex(firstEnabledIndex >= 0 ? firstEnabledIndex : 0);
      if (firstEnabledIndex >= 0) {
        menuItems.current[firstEnabledIndex]?.focus();
      }
    }
  }, [open, items, prefetchedItems]);

  const handleActionSelection = (value: string) => {
    setEnabledItem(value);
  };

  const handleItemHover = async (item: MenuItem) => {
    if (item.prefetch && !prefetchedItems.has(item.id)) {
      try {
        await item.prefetch();
        setPrefetchedItems((prev) => new Set([...prev, item.id]));
      } catch (error) {
        console.error(`Failed to prefetch data for ${item.id}:`, error);
      }
    }
  };

  // Reason for useEffect, https://github.com/radix-ui/primitives/issues/2122
  useEffect(() => {
    if (open) {
      // Pushing the change to the end of the call stack
      const timer = setTimeout(() => {
        document.body.style.pointerEvents = "";
      }, 0);

      return () => clearTimeout(timer);
    }
    document.body.style.pointerEvents = "auto";
  }, [open]);

  return (
    <Popover open={open} onOpenChange={setOpen} modal>
      <PopoverTrigger asChild>
        {children ?? <TableActionPopoverDefaultTrigger onClick={(e) => e.stopPropagation()} />}
      </PopoverTrigger>
      <PopoverContent
        className="min-w-60 max-w-full bg-gray-1 dark:bg-black drop-shadow-2xl transform-gpu border-gray-6 rounded-lg p-0"
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
                <button
                  type="button"
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
                  onMouseEnter={() => handleItemHover(item)}
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
                </button>
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

export const TableActionPopoverDefaultTrigger = forwardRef<
  HTMLButtonElement,
  { onClick?: (e: React.MouseEvent) => void } & React.ComponentProps<typeof Button> &
    ReactLoadableProps
>(({ onClick, ...props }, ref) => {
  // Filter out React Loadable props that shouldn't be passed to DOM elements
  const { isLoading, pastDelay, timedOut, retry, error, ...buttonProps } = props;

  return (
    <Button
      ref={ref}
      variant="outline"
      className="size-5 [&_svg]:size-3 rounded"
      onClick={onClick}
      {...buttonProps}
    >
      <Dots className="group-hover:text-gray-12 text-gray-11" iconSize="sm-regular" />
    </Button>
  );
});

TableActionPopoverDefaultTrigger.displayName = "TableActionPopoverDefaultTrigger";
