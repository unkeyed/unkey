"use client";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { toast } from "@/components/ui/toaster";
import { Clipboard, ClipboardCheck, InputSearch, PenWriting3 } from "@unkey/icons";
import Link from "next/link";
import { type KeyboardEvent, type PropsWithChildren, useEffect, useRef, useState } from "react";
import { useRatelimitLogsContext } from "../../../../context/logs";
import { useFilters } from "../../../../hooks/use-filters";

type Props = {
  identifier: string;
};

export const TableActionPopover = ({ children, identifier }: PropsWithChildren<Props>) => {
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [focusIndex, setFocusIndex] = useState(0);
  const menuItems = useRef<HTMLDivElement[]>([]);
  const { filters, updateFilters } = useFilters();
  const { namespaceId } = useRatelimitLogsContext();

  useEffect(() => {
    if (open) {
      setFocusIndex(0);
      menuItems.current[0]?.focus();
    }
  }, [open]);

  const handleFilterClick = (e: React.MouseEvent | KeyboardEvent) => {
    e.stopPropagation();
    const newFilter = {
      id: crypto.randomUUID(),
      field: "identifiers" as const,
      operator: "is" as const,
      value: identifier,
    };
    const existingFilters = filters.filter(
      (f) => !(f.field === "identifiers" && f.value === identifier),
    );
    updateFilters([...existingFilters, newFilter]);
    setOpen(false);
  };

  const handleCopy = (e: React.MouseEvent | KeyboardEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(identifier);
    toast.success("Copied to clipboard", {
      description: identifier,
    });
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
    setOpen(false);
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    e.stopPropagation();

    const activeElement = document.activeElement;
    const currentIndex = menuItems.current.findIndex((item) => item === activeElement);

    switch (e.key) {
      case "Tab":
        e.preventDefault();
        if (!e.shiftKey) {
          setFocusIndex((currentIndex + 1) % 3);
          menuItems.current[(currentIndex + 1) % 3]?.focus();
        } else {
          setFocusIndex((currentIndex - 1 + 3) % 3);
          menuItems.current[(currentIndex - 1 + 3) % 3]?.focus();
        }
        break;
      case "j":
      case "ArrowDown":
        e.preventDefault();
        setFocusIndex((currentIndex + 1) % 3);
        menuItems.current[(currentIndex + 1) % 3]?.focus();
        break;
      case "k":
      case "ArrowUp":
        e.preventDefault();
        setFocusIndex((currentIndex - 1 + 3) % 3);
        menuItems.current[(currentIndex - 1 + 3) % 3]?.focus();
        break;
      case "Escape":
        e.preventDefault();
        setOpen(false);
        break;
      case "Enter":
      case "ArrowRight":
      case "l":
      case " ":
        e.preventDefault();
        if (activeElement === menuItems.current[0]) {
          handleCopy(e);
        } else if (activeElement === menuItems.current[2]) {
          handleFilterClick(e);
        }
        break;
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger onClick={(e) => e.stopPropagation()} asChild>
        <div>{children}</div>
      </PopoverTrigger>
      <PopoverContent
        className="w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg"
        align="start"
        onOpenAutoFocus={(e) => {
          e.preventDefault();
          menuItems.current[0]?.focus();
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
          className="flex flex-col gap-1"
          role="menu"
          onClick={(e) => e.stopPropagation()}
          onKeyDown={handleKeyDown}
        >
          <PopoverHeader />
          {/* biome-ignore lint/a11y/useKeyWithClickEvents: it's okay */}
          <div
            ref={(el) => {
              if (el) {
                menuItems.current[0] = el;
              }
            }}
            role="menuitem"
            tabIndex={focusIndex === 0 ? 0 : -1}
            className="flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer
            hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none focus:bg-gray-3"
            onClick={handleCopy}
          >
            <span className="text-[13px] text-accent-12 font-medium">Copy identifier</span>
            {copied ? <ClipboardCheck /> : <Clipboard />}
          </div>
          <Link
            href={`/ratelimits/${namespaceId}/overrides?identifier=${identifier}`}
            tabIndex={-1}
            onClick={(e) => e.stopPropagation()}
          >
            <div
              ref={(el) => {
                if (el) {
                  menuItems.current[1] = el;
                }
              }}
              role="menuitem"
              tabIndex={focusIndex === 1 ? 0 : -1}
              className="flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer
              hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none focus:bg-gray-3"
            >
              <span className="text-[13px] text-accent-12 font-medium">Override</span>
              <PenWriting3 />
            </div>
          </Link>
          {/* biome-ignore lint/a11y/useKeyWithClickEvents: it's okay */}
          <div
            ref={(el) => {
              if (el) {
                menuItems.current[2] = el;
              }
            }}
            role="menuitem"
            tabIndex={focusIndex === 2 ? 0 : -1}
            className="flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer
            hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none focus:bg-gray-3"
            onClick={handleFilterClick}
          >
            <span className="text-[13px] text-accent-12 font-medium">Filter for identifier</span>
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
      <span className="text-gray-9 text-[13px]">Actions</span>
    </div>
  );
};
