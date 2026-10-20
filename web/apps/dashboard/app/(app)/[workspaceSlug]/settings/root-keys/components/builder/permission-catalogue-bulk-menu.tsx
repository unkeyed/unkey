"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { IconChevronDownOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import type { BulkOption } from "./lib/policy";

const BULK_OPTIONS: { id: BulkOption; label: string }[] = [
  { id: "read", label: "Read only" },
  { id: "read-write", label: "Read & write" },
  { id: "full", label: "Full control" },
  { id: "clear", label: "Clear all" },
];

type PermissionCatalogueBulkMenuProps = {
  onSelect: (option: BulkOption) => void;
};

export function PermissionCatalogueBulkMenu({ onSelect }: PermissionCatalogueBulkMenuProps) {
  const [open, setOpen] = useState(false);

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger
        render={
          <Button variant="outline" size="md" className="shrink-0">
            Select all…
            <IconChevronDownOutline18 className="text-gray-9" />
          </Button>
        }
      />
      <DropdownMenuContent align="end" className="w-48">
        {BULK_OPTIONS.map((option) => (
          <DropdownMenuItem
            key={option.id}
            className="cursor-pointer"
            onClick={() => {
              setOpen(false);
              onSelect(option.id);
            }}
          >
            <span className="text-sm font-medium text-gray-12">{option.label}</span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
