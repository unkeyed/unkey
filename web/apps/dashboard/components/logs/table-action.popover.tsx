import {
  Button,
  Popover,
  PopoverContent,
  PopoverTrigger,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Route } from "next";
import Link from "next/link";
import { IconDotsOutline18 } from "nucleo-ui-outline-18";
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
  // Navigation items pass href so they render as links: new tab, middle click
  // and prefetch keep working.
  href?: Route;
  className?: string;
  // Pass a function when the disabled or tooltip state depends on a moving
  // value (e.g. wall-clock time) so it is re-evaluated each render instead
  // of being captured at item-build time.
  disabled?: boolean | (() => boolean);
  divider?: boolean;
  ActionComponent?: FC<ActionComponentProps>;
  prefetch?: () => Promise<void>;
  tooltip?: string | (() => string | undefined);
};

const isItemDisabled = (item: MenuItem): boolean =>
  typeof item.disabled === "function" ? item.disabled() : Boolean(item.disabled);

const itemTooltip = (item: MenuItem): string | undefined =>
  typeof item.tooltip === "function" ? item.tooltip() : item.tooltip;

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
  const [prefetchedItems, setPrefetchedItems] = useState<Set<string>>(new Set());
  const menuItems = useRef<(HTMLElement | null)[]>([]);

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

      const firstEnabledIndex = items.findIndex((item) => !isItemDisabled(item));
      if (firstEnabledIndex >= 0) {
        menuItems.current[firstEnabledIndex]?.focus();
      }
    }
  }, [open, items, prefetchedItems]);

  useEffect(() => {
    if (enabledItem && !items.some((item) => item.id === enabledItem)) {
      setEnabledItem(undefined);
    }
  }, [items, enabledItem]);

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

  return (
    <>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            (children ?? (
              <TableActionPopoverDefaultTrigger onClick={(e) => e.stopPropagation()} />
            )) as React.ReactElement
          }
        />
        <PopoverContent
          className="min-w-60 max-w-full bg-gray-1 dark:bg-black drop-shadow-2xl transform-gpu border-gray-6 rounded-lg p-0"
          align={align}
          initialFocus={() => {
            const firstEnabledIndex = items.findIndex((item) => !isItemDisabled(item));
            return firstEnabledIndex >= 0 ? (menuItems.current[firstEnabledIndex] ?? false) : false;
          }}
          finalFocus={false}
        >
          {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
          <div role="menu" onClick={(e) => e.stopPropagation()} className="py-2">
            {items.map((item, index) => {
              const disabled = isItemDisabled(item);
              const tooltip = itemTooltip(item);
              const itemProps = {
                role: "menuitem" as const,
                "aria-disabled": disabled,
                tabIndex: disabled ? -1 : 0,
                className: cn(
                  "flex w-full items-center px-2 py-1.5 gap-3 rounded-lg group",
                  !disabled &&
                    "cursor-pointer hover:bg-gray-3 data-popup-open:bg-gray-3 focus:outline-hidden focus:bg-gray-3",
                  disabled && "cursor-not-allowed opacity-50",
                  item.className,
                ),
                ref: (element: HTMLElement | null) => {
                  menuItems.current[index] = element;
                },
                onMouseEnter: () => handleItemHover(item),
                onClick: (e: React.MouseEvent<Element, MouseEvent>) => {
                  if (disabled) {
                    return;
                  }
                  item.onClick?.(e);
                  setEnabledItem(item.id);
                  setOpen(false);
                },
              };
              const body = (
                <>
                  <div className="text-gray-9 group-hover:text-gray-12 group-focus:text-gray-12">
                    {item.icon}
                  </div>
                  <span className="text-[13px] font-normal">{item.label}</span>
                </>
              );
              const control =
                item.href && !disabled ? (
                  <Link href={item.href} {...itemProps}>
                    {body}
                  </Link>
                ) : (
                  <button type="button" {...itemProps}>
                    {body}
                  </button>
                );
              return (
                <div key={item.id}>
                  <div className="px-2">
                    {tooltip ? (
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger render={control} />
                          <TooltipContent className="z-[9998]">{tooltip}</TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    ) : (
                      control
                    )}
                  </div>
                  {item.divider && <div aria-hidden className="h-px bg-grayA-3 w-full my-2" />}
                </div>
              );
            })}
          </div>
        </PopoverContent>
      </Popover>
      {/* Render ActionComponents outside the Popover so they persist when popover closes */}
      {items.map(
        (item) =>
          item.ActionComponent &&
          enabledItem === item.id && (
            <div
              key={item.id}
              onClick={(e) => e.stopPropagation()}
              onKeyDown={(e) => e.stopPropagation()}
            >
              <item.ActionComponent
                isOpen
                onClose={() => {
                  handleActionSelection("none");
                }}
              />
            </div>
          ),
      )}
    </>
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
      className="size-5 [&_svg]:size-3 rounded-sm"
      onClick={onClick}
      aria-label="Open actions"
      {...buttonProps}
    >
      <IconDotsOutline18 className="group-hover:text-gray-12 text-gray-11" />
    </Button>
  );
});

TableActionPopoverDefaultTrigger.displayName = "TableActionPopoverDefaultTrigger";
