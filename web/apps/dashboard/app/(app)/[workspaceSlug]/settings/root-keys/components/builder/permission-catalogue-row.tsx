"use client";

import { Checkbox } from "@unkey/ui";
import { useId } from "react";
import { type Action, type PermissionRow, rowOffers } from "./lib/catalogue.types";

export const ACTION_LABELS: Record<Action, string> = {
  read: "Read",
  write: "Write",
  delete: "Delete",
  verify: "Verify",
  decrypt: "Decrypt",
  limit: "Limit",
};

type PermissionCatalogueRowProps = {
  row: PermissionRow;
  actions: readonly Action[];
  columns: readonly Action[];
  onToggle: (action: Action, selected: boolean) => void;
};

export function PermissionCatalogueRow({
  row,
  actions,
  columns,
  onToggle,
}: PermissionCatalogueRowProps) {
  const id = useId();

  return (
    <div className="flex items-center justify-between gap-4 py-2">
      <span className="min-w-0 flex-1 truncate text-sm text-gray-12">{row.label}</span>
      <div className="flex items-center gap-4 shrink-0">
        {columns.map((action) =>
          rowOffers(row, action) ? (
            <div key={action} className="flex items-center gap-2">
              <Checkbox
                id={`${id}-${action}`}
                size="md"
                checked={actions.includes(action)}
                onCheckedChange={(next) => onToggle(action, next === true)}
              />
              <label
                htmlFor={`${id}-${action}`}
                className="cursor-pointer select-none text-xs text-gray-12"
              >
                {ACTION_LABELS[action]}
              </label>
            </div>
          ) : (
            <div key={action} className="invisible flex items-center gap-2" aria-hidden="true">
              <span className="size-3.5" />
              <span className="text-xs">{ACTION_LABELS[action]}</span>
            </div>
          ),
        )}
      </div>
    </div>
  );
}
