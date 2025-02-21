"use client";

import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { toast } from "@/components/ui/toaster";
import { Clone, Layers3, PenWriting3 } from "@unkey/icons";
import Link from "next/link";
import {
  type KeyboardEvent,
  type PropsWithChildren,
  useEffect,
  useRef,
  useState,
} from "react";
import { useFilters } from "../../../../../hooks/use-filters";
import type { OverrideDetails } from "../../../logs-table";
import { IdentifierDialog } from "./identifier-dialog";

type Props = {
  identifier: string;
  namespaceId: string;
  overrideDetails?: OverrideDetails | null;
};

export const TableActionPopover = ({
  children,
  identifier,
  namespaceId,
  overrideDetails,
}: PropsWithChildren<Props>) => {
  const [isModalOpen, setIsModalOpen] = useState(false);

  const [open, setOpen] = useState(false);
  const [focusIndex, setFocusIndex] = useState(0);
  const menuItems = useRef<HTMLDivElement[]>([]);
  const { filters, updateFilters } = useFilters();

  const timeFilters = filters.filter((f) =>
    ["startTime", "endTime", "since"].includes(f.field)
  );

  const getTimeParams = () => {
    const params = new URLSearchParams({
      identifier: `contains:${identifier}`,
    });

    // Only add time parameters if they exist
    const timeMap = {
      startTime: timeFilters.find((f) => f.field === "startTime")?.value,
      endTime: timeFilters.find((f) => f.field === "endTime")?.value,
      since: timeFilters.find((f) => f.field === "since")?.value,
    };

    Object.entries(timeMap).forEach(([key, value]) => {
      if (value) {
        params.append(key, value.toString());
      }
    });

    return params.toString();
  };

  useEffect(() => {
    if (open) {
      setFocusIndex(0);
      menuItems.current[0]?.focus();
    }
  }, [open]);

  const handleEditClick = (e: React.MouseEvent | KeyboardEvent) => {
    e.stopPropagation();
    setOpen(false);
    setIsModalOpen(true);
  };

  const handleFilterClick = (e: React.MouseEvent | KeyboardEvent) => {
    e.stopPropagation();
    const newFilter = {
      id: crypto.randomUUID(),
      field: "identifiers" as const,
      operator: "is" as const,
      value: identifier,
    };
    const existingFilters = filters.filter(
      (f) => !(f.field === "identifiers" && f.value === identifier)
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
    setOpen(false);
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    e.stopPropagation();

    const activeElement = document.activeElement;
    const currentIndex = menuItems.current.findIndex(
      (item) => item === activeElement
    );

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
    <>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger onClick={(e) => e.stopPropagation()} asChild>
          <div>{children}</div>
        </PopoverTrigger>
        <PopoverContent
          className="w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg"
          align="end"
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
            className="flex flex-col gap-2"
            role="menu"
            onClick={(e) => e.stopPropagation()}
            onKeyDown={handleKeyDown}
          >
            <Link
              href={`/ratelimits/${namespaceId}/logs?${getTimeParams()}`}
              tabIndex={-1}
              onClick={(e) => e.stopPropagation()}
            >
              <div
                ref={(el) => {
                  if (el) {
                    menuItems.current[0] = el;
                  }
                }}
                role="menuitem"
                tabIndex={focusIndex === 0 ? 0 : -1}
                className="flex w-full items-center px-2 py-1.5 gap-3 rounded-lg group cursor-pointer
            hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none focus:bg-gray-3"
              >
                <Layers3 size="md-regular" />
                <span className="text-[13px] text-accent-12 font-medium">
                  Go to logs
                </span>
              </div>
            </Link>
            {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
            <div
              ref={(el) => {
                if (el) {
                  menuItems.current[1] = el;
                }
              }}
              role="menuitem"
              tabIndex={focusIndex === 1 ? 0 : -1}
              className="flex w-full items-center px-2 py-1.5 gap-3 rounded-lg group cursor-pointer
            hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none focus:bg-gray-3"
              onClick={handleCopy}
            >
              <Clone size="md-regular" />
              <span className="text-[13px] text-accent-12 font-medium">
                Copy identifier
              </span>
            </div>

            {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
            <div
              ref={(el) => {
                if (el) {
                  menuItems.current[2] = el;
                }
              }}
              role="menuitem"
              tabIndex={focusIndex === 2 ? 0 : -1}
              className="flex w-full items-center px-2 py-1.5 gap-3 rounded-lg group cursor-pointer
            hover:bg-orange-3 data-[state=open]:bg-orange-3 focus:outline-none focus:bg-orange-3 text-orange-11"
              onClick={handleEditClick}
            >
              <PenWriting3 size="md-regular" />
              <span className="text-[13px] font-medium">
                Override Identifier
              </span>
            </div>
          </div>
        </PopoverContent>
      </Popover>
      <IdentifierDialog
        overrideDetails={overrideDetails}
        namespaceId={namespaceId}
        identifier={identifier}
        isModalOpen={isModalOpen}
        onOpenChange={setIsModalOpen}
      />
    </>
  );
};
