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
import { ACTIONS, type Action, READ_ACTIONS, READ_WRITE_ACTIONS } from "./lib/catalogue.types";

const BULK_OPTIONS: { id: string; label: string; actions: readonly Action[] }[] = [
  { id: "read", label: "Read only", actions: READ_ACTIONS },
  { id: "read-write", label: "Read & write", actions: READ_WRITE_ACTIONS },
  { id: "full", label: "Full control", actions: ACTIONS },
  { id: "clear", label: "Clear all", actions: [] },
];

type PermissionCatalogueBulkMenuProps = {
  offered: readonly Action[];
  onSelect: (actions: readonly Action[]) => void;
};

export function PermissionCatalogueBulkMenu({
  offered,
  onSelect,
}: PermissionCatalogueBulkMenuProps) {
  const [open, setOpen] = useState(false);
  const reach = (actions: readonly Action[]) =>
    actions.filter((action) => offered.includes(action)).length;
  const options = BULK_OPTIONS.filter(
    (option, index) =>
      option.actions.length === 0 ||
      reach(option.actions) > reach(BULK_OPTIONS[index - 1]?.actions ?? []),
  );

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
        {options.map((option) => (
          <DropdownMenuItem
            key={option.id}
            className="cursor-pointer"
            onClick={() => {
              setOpen(false);
              onSelect(option.actions);
            }}
          >
            <span className="text-sm font-medium text-gray-12">{option.label}</span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
