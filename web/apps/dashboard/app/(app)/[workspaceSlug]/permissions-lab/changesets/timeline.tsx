"use client";

/**
 * Right column of the changesets concept: every changeset from the shared lab
 * store rendered newest-first as a vertical timeline. Staged changesets can be
 * applied or discarded, applied ones reverted; reverted ones stay as history.
 */

import { ChevronDown } from "@unkey/icons";
import { Button, Empty, cn, toast } from "@unkey/ui";
import { useState } from "react";
import { type Changeset, usePermissionsLab } from "../lib/store";
import { DiffLine } from "./diff-line";

const STATUS_PILL: Record<Changeset["status"], string> = {
  staged: "bg-warningA-3 text-warning-11",
  applied: "bg-successA-3 text-success-11",
  reverted: "bg-grayA-3 text-gray-11",
};

const STATUS_DOT: Record<Changeset["status"], string> = {
  staged: "bg-warning-9",
  applied: "bg-success-9",
  reverted: "bg-gray-8",
};

function TimelineEntry({ changeset }: { changeset: Changeset }) {
  const lab = usePermissionsLab();
  const [expanded, setExpanded] = useState(changeset.status === "staged");
  const opCount = changeset.ops.length;

  return (
    <li className="relative">
      <span
        aria-hidden="true"
        className={cn(
          "absolute -left-[26px] top-2 size-3 rounded-full",
          STATUS_DOT[changeset.status],
        )}
      />
      <div
        className={cn(
          "flex flex-col gap-2 rounded-lg border border-grayA-4 p-3",
          changeset.status === "reverted" && "opacity-70",
        )}
      >
        <div className="flex items-center gap-2">
          <span
            className={cn(
              "rounded px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide",
              STATUS_PILL[changeset.status],
            )}
          >
            {changeset.status}
          </span>
          <span className="min-w-0 flex-1 truncate text-sm font-medium text-gray-12">
            {changeset.title}
          </span>
          <button
            type="button"
            onClick={() => setExpanded(!expanded)}
            aria-expanded={expanded}
            className="flex items-center gap-1 text-xs text-gray-10 transition-colors hover:text-gray-12"
          >
            {opCount} {opCount === 1 ? "change" : "changes"}
            <ChevronDown
              iconSize="sm-regular"
              className={cn("transition-transform", expanded && "rotate-180")}
            />
          </button>
        </div>

        {expanded && (
          <div className="flex flex-col gap-1.5">
            {changeset.ops.map((op) => (
              <DiffLine key={JSON.stringify(op)} op={op} />
            ))}
          </div>
        )}

        {changeset.status === "staged" && (
          <div className="flex gap-2">
            <Button
              onClick={() => {
                lab.apply(changeset.id);
                toast.success(`Applied "${changeset.title}"`);
              }}
            >
              Apply
            </Button>
            <Button
              variant="ghost"
              color="danger"
              onClick={() => {
                lab.discard(changeset.id);
                toast(`Discarded "${changeset.title}"`);
              }}
            >
              Discard
            </Button>
          </div>
        )}
        {changeset.status === "applied" && (
          <div className="flex">
            <Button
              variant="outline"
              onClick={() => {
                lab.revert(changeset.id);
                toast(`Reverted "${changeset.title}"`);
              }}
            >
              Revert
            </Button>
          </div>
        )}
      </div>
    </li>
  );
}

export function Timeline() {
  const { state } = usePermissionsLab();

  if (state.changesets.length === 0) {
    return (
      <Empty>
        <Empty.Title>No changesets yet</Empty.Title>
        <Empty.Description>
          Stage a draft from the composer and it will show up here, ready to apply.
        </Empty.Description>
      </Empty>
    );
  }

  // state.changesets is already newest-first: stage() prepends.
  return (
    <ol className="relative ml-1.5 flex flex-col gap-4 border-l border-grayA-5 pl-5">
      {state.changesets.map((changeset) => (
        <TimelineEntry key={changeset.id} changeset={changeset} />
      ))}
    </ol>
  );
}
