"use client";

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import { cn } from "@/lib/utils";
import { Check } from "@unkey/icons";
import type { IconProps } from "@unkey/icons";
import { Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { type ComponentType, type ReactNode, useState } from "react";

export type CrumbPopoverItem = {
  id: string;
  label: string;
  href?: string;
  onClick?: () => void;
  badge?: ReactNode;
};

export type CrumbPopoverFooter = {
  icon: ComponentType<IconProps>;
  label: string;
  href?: string;
  onClick?: () => void;
};

type CrumbPopoverProps = {
  items: CrumbPopoverItem[];
  currentId?: string;
  searchPlaceholder: string;
  emptyText: string;
  footer: CrumbPopoverFooter;
  children: ReactNode;
};

/**
 * Supabase-style switcher popover used by each crumb in the v2b top
 * header. Search input → item list with a checkmark on the current
 * selection → separator → a single footer action row ("+ New X").
 */
export function CrumbPopover({
  items,
  currentId,
  searchPlaceholder,
  emptyText,
  footer,
  children,
}: CrumbPopoverProps) {
  const [open, setOpen] = useState(false);
  const router = useRouter();

  const selectItem = (item: CrumbPopoverItem) => {
    setOpen(false);
    if (item.href) {
      router.push(item.href);
    } else if (item.onClick) {
      item.onClick();
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent align="start" className="w-64 p-0" sideOffset={8}>
        <Command
          // cmdk's default filter treats label values as haystack; the id
          // is stable and non-user-facing, so matching on `label` is what
          // we want. Ensuring the filter also looks inside `value` (which
          // cmdk sets from `CommandItem value=`) covers the case.
          filter={(value, search) => {
            if (!search) {
              return 1;
            }
            return value.toLowerCase().includes(search.toLowerCase()) ? 1 : 0;
          }}
        >
          <CommandInput placeholder={searchPlaceholder} />
          <CommandList>
            <CommandEmpty className="py-6">{emptyText}</CommandEmpty>
            <CommandGroup>
              {items.map((item) => {
                const isCurrent = item.id === currentId;
                return (
                  <CommandItem
                    key={item.id}
                    value={item.label}
                    onSelect={() => selectItem(item)}
                    className="gap-2"
                  >
                    <span className="flex-1 truncate">{item.label}</span>
                    {item.badge ? <span className="shrink-0">{item.badge}</span> : null}
                    <Check
                      iconSize="sm-regular"
                      className={cn(
                        "size-3.5 shrink-0 text-accent-12",
                        isCurrent ? "opacity-100" : "opacity-0",
                      )}
                    />
                  </CommandItem>
                );
              })}
            </CommandGroup>
            <CommandSeparator />
            <CommandGroup>
              <CrumbPopoverFooterRow
                footer={footer}
                onSelect={() => {
                  setOpen(false);
                  if (footer.onClick) {
                    footer.onClick();
                  }
                }}
              />
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

function CrumbPopoverFooterRow({
  footer,
  onSelect,
}: {
  footer: CrumbPopoverFooter;
  onSelect: () => void;
}) {
  const Icon = footer.icon;
  const body = (
    <>
      <Icon iconSize="sm-regular" className="size-3.5 shrink-0 text-accent-11" />
      <span className="flex-1 truncate">{footer.label}</span>
    </>
  );

  if (footer.href) {
    return (
      <CommandItem asChild value={`__footer__${footer.label}`}>
        <Link href={footer.href} onClick={onSelect} className="gap-2">
          {body}
        </Link>
      </CommandItem>
    );
  }

  return (
    <CommandItem value={`__footer__${footer.label}`} onSelect={onSelect} className="gap-2">
      {body}
    </CommandItem>
  );
}
