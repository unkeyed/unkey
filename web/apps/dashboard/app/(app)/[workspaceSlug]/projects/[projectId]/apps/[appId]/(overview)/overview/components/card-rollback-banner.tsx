"use client";

import { ArrowDottedRotateAnticlockwise } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useProductionCard } from "./production-card-context";

export function ProductionCardRollbackBanner() {
  const { openUndo } = useProductionCard();
  return (
    <div className="relative z-0 -mb-3 flex items-center justify-between gap-3 rounded-t-[14px] border border-b-0 border-warning-6 bg-warning-3 px-4 pt-2.5 pb-5">
      <div className="flex items-center gap-1.5 text-[13px] min-w-0">
        <ArrowDottedRotateAnticlockwise
          iconSize="sm-regular"
          className="text-warning-11 shrink-0"
        />
        <span className="font-medium text-accent-12 shrink-0">Rolled back</span>
        <span className="text-gray-12 truncate">
          — live domains do not get reassigned until you undo
        </span>
      </div>
      <Button variant="outline" size="sm" onClick={openUndo} className="shrink-0 bg-gray-1">
        Undo Rollback
      </Button>
    </div>
  );
}
