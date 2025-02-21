"use client";
import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { CaretRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useRouter } from "next/navigation";
import { type PropsWithChildren, useEffect, useRef, useState } from "react";

export type QuickNavItem = {
  id: string;
  label: React.ReactNode;
  shortcut?: string;
  href?: string;
  onClick?: () => void;
  className?: string;
  itemClassName?: string;
  hideRightIcon?: boolean;
};

type QuickNavPopoverProps = {
  items: QuickNavItem[];
  title?: string;
  shortcutKey?: string;
  onItemSelect?: (item: QuickNavItem) => void;
};

export const QuickNavPopover = ({
  children,
  items,
  title = "Navigate to...",
  shortcutKey = "f",
  onItemSelect,
}: PropsWithChildren<QuickNavPopoverProps>) => {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null);

  useKeyboardShortcut({ key: shortcutKey, ctrl: true }, () => {
    setOpen((prev) => !prev);
  });

  const handleItemSelect = (item: QuickNavItem) => {
    setOpen(false);
    if (onItemSelect) {
      onItemSelect(item);
      return
    }
    if (item.onClick) {
      item.onClick();
      return;
    }
    if (item.href) {
      router.push(item.href);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!open) {
      return;
    }
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setFocusedIndex((prev) => (prev === null ? 0 : (prev + 1) % items.length));
        break;
      case "ArrowUp":
        e.preventDefault();
        setFocusedIndex((prev) =>
          prev === null ? items.length - 1 : (prev - 1 + items.length) % items.length,
        );
        break;
      case "Enter":
      case "ArrowRight":
        e.preventDefault();
        if (focusedIndex !== null) {
          const selectedItem = items[focusedIndex];
          handleItemSelect(selectedItem);
        }
        break;
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger>{children}</PopoverTrigger>
      <PopoverContent
        className={cn("w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg")}
        align="start"
        onKeyDown={handleKeyDown}
      >
        <div className="flex flex-col gap-2">
          <PopoverHeader title={title} shortcutKey={shortcutKey.toUpperCase()} />
          {items.map((item, index) => (
            <PopoverItem
              key={item.id}
              {...item}
              isFocused={focusedIndex === index}
              onSelect={() => handleItemSelect(item)}
            />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = ({
  title,
  shortcutKey,
}: {
  title: string;
  shortcutKey: string;
}) => (
  <div className="flex w-full justify-between items-center px-2 py-1">
    <span className="text-gray-9 text-[13px]">{title}</span>
    <KeyboardButton shortcut={shortcutKey} modifierKey="⌃" />
  </div>
);

type PopoverItemProps = QuickNavItem & {
  isFocused?: boolean;
  isActive?: boolean;
  onSelect: () => void;
};

const PopoverItem = ({
  label,
  isFocused,
  onSelect,
  className,
  itemClassName,
  hideRightIcon,
}: PopoverItemProps) => {
  const itemRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (isFocused && itemRef.current) {
      itemRef.current.focus();
    }
  }, [isFocused]);

  return (
    <button
      type="button"
      ref={itemRef}
      className={cn(
        "flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer",
        "hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none",
        isFocused && "bg-gray-3",
        itemClassName,
      )}
      tabIndex={0}
      onClick={onSelect}
    >
      <div className={cn("flex gap-2 items-center", className)}>
        <span className="text-[13px] text-accent-12 font-medium">{label}</span>
      </div>
      {!hideRightIcon && (
        <div className="flex items-center gap-1.5">
          <Button variant="ghost" size="icon" tabIndex={-1} className="size-5 [&_svg]:size-2">
            <CaretRight className="text-gray-7 group-hover:text-gray-10" />
          </Button>
        </div>
      )}
    </button>
  );
};
