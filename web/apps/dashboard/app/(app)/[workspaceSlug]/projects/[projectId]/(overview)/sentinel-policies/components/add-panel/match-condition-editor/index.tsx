"use client";

import { cn } from "@/lib/utils";
import { Plus } from "@unkey/icons";
import { Button, Separator } from "@unkey/ui";
import { Fragment } from "react";
import type { MatchConditionFormValues } from "../schema";
import { MatchConditionCard } from "./condition-card";

export const MAX_MATCH_CONDITIONS = 10;

export function MatchConditionEditorBody({
  conditions,
  onChange,
}: {
  conditions: MatchConditionFormValues[];
  onChange: (conditions: MatchConditionFormValues[]) => void;
}) {
  const atCap = conditions.length >= MAX_MATCH_CONDITIONS;
  const hasConditions = conditions.length > 0;
  return (
    <div className="flex flex-col">
      {hasConditions && (
        <div className="flex flex-col gap-8 pt-3">
          {conditions.map((cond, index) => (
            <Fragment key={cond.id}>
              <MatchConditionCard
                condition={cond}
                onChange={(updated) =>
                  onChange(conditions.map((c) => (c.id === updated.id ? updated : c)))
                }
                onDelete={(id) => {
                  onChange(conditions.filter((c) => c.id !== id));
                }}
              />
              {index < conditions.length - 1 && <Separator className="bg-gray-2" />}
            </Fragment>
          ))}
        </div>
      )}
      <div className={cn("flex items-center gap-3", hasConditions && "pt-6")}>
        <Button
          type="button"
          variant="outline"
          size="md"
          className="font-medium"
          disabled={atCap}
          onClick={() =>
            onChange([
              ...conditions,
              { id: crypto.randomUUID(), type: "path", mode: "exact", value: "" },
            ])
          }
        >
          <Plus iconSize="sm-regular" />
          {conditions.length === 0 ? "Add First Condition" : "Add Condition"}
        </Button>
        <span className="text-[12px] text-gray-11">
          {conditions.length} / {MAX_MATCH_CONDITIONS}
          {atCap && " · maximum reached"}
        </span>
      </div>
    </div>
  );
}

