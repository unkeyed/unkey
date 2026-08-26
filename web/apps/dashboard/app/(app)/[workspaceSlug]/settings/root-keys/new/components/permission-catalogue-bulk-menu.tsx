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
import { ACTIONS, type Action } from "../lib/catalogue.types";

const BULK_OPTIONS: { id: string; label: string; actions: readonly Action[] }[] = [
  {
    id: "read",
    label: "Read only",
    actions: [
      "read_project",
      "read_app",
      "read_environment",
      "read_deployment",
      "read_deployment_logs",
      "read_domain",
      "read_environment_variable",
      "read_gateway_logs",
      "read_gateway_policy",
      "read_identity",
      "read_keyspace",
      "read_keyspace_logs",
      "read_key",
      "read_ratelimit_namespace",
      "read_ratelimit_logs",
      "read_ratelimit_override",
      "read_role",
      "read_permission",
      "read_github_app",
    ],
  },
  {
    id: "read-write",
    label: "Read & write",
    actions: [
      "read_project",
      "write_project",
      "read_app",
      "write_app",
      "read_environment",
      "write_environment",
      "read_deployment",
      "write_deployment",
      "read_deployment_logs",
      "read_domain",
      "write_domain",
      "read_environment_variable",
      "write_environment_variable",
      "read_gateway_logs",
      "read_gateway_policy",
      "write_gateway_policy",
      "read_identity",
      "write_identity",
      "read_keyspace",
      "write_keyspace",
      "read_keyspace_logs",
      "read_key",
      "write_key",
      "read_ratelimit_namespace",
      "write_ratelimit_namespace",
      "read_ratelimit_logs",
      "read_ratelimit_override",
      "write_ratelimit_override",
      "read_role",
      "write_role",
      "read_permission",
      "write_permission",
      "read_github_app",
      "write_github_app",
    ],
  },
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
