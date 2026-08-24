"use client";

import { Checkbox } from "@unkey/ui";
import { useId } from "react";
import { ACTIONS, type Action, type PermissionRow } from "../lib/catalogue.types";

const ACTION_LABELS: Record<Action, string> = {
  read: "Read",
  write: "Write",
  delete: "Delete",
};

type PermissionCatalogueRowProps = {
  row: PermissionRow;
  actions: readonly Action[];
  onToggle: (action: Action, selected: boolean) => void;
};

export function PermissionCatalogueRow({ row, actions, onToggle }: PermissionCatalogueRowProps) {
  const id = useId();

  return (
    <div className="flex items-center justify-between gap-4 py-2">
      <span className="min-w-0 truncate text-[13px] text-accent-12">{row.label}</span>
      <div className="flex items-center gap-4 shrink-0">
        {ACTIONS.map((action) => (
          <div key={action} className="flex items-center gap-2">
            <Checkbox
              id={`${id}-${action}`}
              size="md"
              checked={actions.includes(action)}
              onCheckedChange={(next) => onToggle(action, next === true)}
            />
            <label
              htmlFor={`${id}-${action}`}
              className="text-xs text-accent-12 cursor-pointer select-none"
            >
              {ACTION_LABELS[action]}
            </label>
          </div>
        ))}
      </div>
    </div>
  );
}
