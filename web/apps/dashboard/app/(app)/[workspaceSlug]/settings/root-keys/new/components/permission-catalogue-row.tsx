"use client";

import { Checkbox } from "@unkey/ui";
import { useId } from "react";
import { ACTIONS, type Action, type PermissionRow } from "../lib/catalogue.types";
import { grantPaths } from "../lib/urn";

const ACTION_LABELS: Record<Action, string> = {
  read: "Read",
  write: "Write",
  delete: "Delete",
};

type PermissionCatalogueRowProps = {
  row: PermissionRow;
  grants: readonly string[];
  actions: readonly Action[];
  onToggle: (action: Action, selected: boolean) => void;
};

export function PermissionCatalogueRow({
  row,
  grants,
  actions,
  onToggle,
}: PermissionCatalogueRowProps) {
  const id = useId();
  const paths = grantPaths(grants);

  return (
    <div className="flex items-center justify-between gap-4 py-2">
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="truncate text-[13px] text-accent-12">{row.label}</span>
        {paths.length === 0 ? null : (
          <span className="whitespace-pre-line break-all font-mono text-xs leading-4 text-gray-9">
            {paths.join("\n")}
          </span>
        )}
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
