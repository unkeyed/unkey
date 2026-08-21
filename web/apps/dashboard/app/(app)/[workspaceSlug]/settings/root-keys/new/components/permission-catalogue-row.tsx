"use client";

import { Checkbox, InfoTooltip } from "@unkey/ui";
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
      <div className="flex flex-col gap-1 min-w-0">
        <span className="text-[13px] text-accent-12">{row.label}</span>
        <InfoTooltip content={row.description} className="max-w-xs">
          <p className="text-xs text-gray-10 truncate text-left">{row.description}</p>
        </InfoTooltip>
      </div>
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
              className="text-xs text-gray-11 cursor-pointer select-none"
            >
              {ACTION_LABELS[action]}
            </label>
          </div>
        ))}
      </div>
    </div>
  );
}
