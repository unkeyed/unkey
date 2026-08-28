"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ChevronDown } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { ACTIONS, type Action, READ_ACTIONS, READ_WRITE_ACTIONS } from "./lib/catalogue.types";

const BULK_OPTIONS: { id: string; label: string; actions: readonly Action[] }[] = [
  { id: "read", label: "Read only", actions: READ_ACTIONS },
  { id: "read-write", label: "Read & write", actions: READ_WRITE_ACTIONS },
  { id: "full", label: "Full control", actions: ACTIONS },
  { id: "clear", label: "Clear all", actions: [] },
];

type PermissionCatalogueBulkMenuProps = {
  onSelect: (actions: readonly Action[]) => void;
};

export function PermissionCatalogueBulkMenu({ onSelect }: PermissionCatalogueBulkMenuProps) {
  const [open, setOpen] = useState(false);

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger
        render={
          <Button variant="outline" size="md" className="shrink-0">
            Select all…
            <ChevronDown className="text-gray-9" />
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
              onSelect(option.actions);
            }}
          >
            <span className="text-accent-12 text-sm font-medium">{option.label}</span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
