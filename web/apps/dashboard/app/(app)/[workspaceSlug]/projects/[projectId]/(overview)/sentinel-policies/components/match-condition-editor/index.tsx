"use client";

import { ChevronDown, Plus } from "@unkey/icons";
import { Button, Separator } from "@unkey/ui";
import { FormDescription } from "@unkey/ui/src/components/form/form-helpers";
import { Fragment, useState } from "react";
import type { MatchConditionFormValues } from "../schema";
import { MatchConditionCard } from "./condition-card";

export function MatchConditionEditor({
  conditions,
  onChange,
}: {
  conditions: MatchConditionFormValues[];
  onChange: (conditions: MatchConditionFormValues[]) => void;
}) {
  const hasConditions = conditions.length > 0;
  const [expanded, setExpanded] = useState(false);

  const addFirstCondition = () => {
    onChange([{ id: crypto.randomUUID(), type: "path", mode: "exact", value: "" }]);
    setExpanded(true);
  };

  if (!expanded && !hasConditions) {
    return (
      <div className="border-t border-grayA-4 px-8 py-4">
        <button
          type="button"
          onClick={addFirstCondition}
          className="flex items-center gap-2 text-[13px] text-gray-11 hover:text-gray-12 transition-colors"
        >
          <Plus iconSize="sm-regular" />
          <span>Add match conditions</span>
          <span className="text-gray-9">to restrict which requests this policy applies to</span>
        </button>
      </div>
    );
  }

  if (!expanded && hasConditions) {
    return (
      <div className="border-t border-grayA-4 px-8 py-4">
        <div className="flex items-center justify-between">
          <button
            type="button"
            onClick={() => setExpanded(true)}
            className="flex items-center gap-2 text-[13px] text-gray-11 hover:text-gray-12 transition-colors"
          >
            <ChevronDown iconSize="sm-regular" className="-rotate-90 transition-transform" />
            <span>
              {conditions.length} match {conditions.length === 1 ? "condition" : "conditions"}{" "}
              configured
            </span>
          </button>
          <button
            type="button"
            onClick={() => {
              onChange([]);
              setExpanded(false);
            }}
            className="text-[12px] text-gray-9 hover:text-gray-12 transition-colors"
          >
            Clear all
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="border-t border-grayA-4">
      <div className="px-8 pt-6 flex items-start justify-between bg-grayA-2">
        <div>
          <span id="match-conditions-label" className="text-gray-11 text-[13px]">
            Match Conditions
          </span>
          <FormDescription
            description={
              <span>
                All conditions must match (<span className="text-gray-12 font-medium">AND</span>{" "}
                logic).
              </span>
            }
            descriptionId="match-conditions-desc"
            errorId="match-conditions-error"
          />
        </div>
        <div className="flex items-center gap-3">
          {hasConditions && (
            <button
              type="button"
              onClick={() => {
                onChange([]);
                setExpanded(false);
              }}
              className="text-[12px] text-gray-9 hover:text-gray-12 transition-colors mt-0.5"
            >
              Clear all
            </button>
          )}
          <button
            type="button"
            onClick={() => setExpanded(false)}
            className="inline-flex items-center justify-center size-7 rounded-md hover:bg-grayA-3 transition-colors text-gray-11 hover:text-gray-12"
            aria-label="Collapse match conditions"
          >
            <ChevronDown iconSize="sm-regular" />
          </button>
        </div>
      </div>

      <div className="flex flex-col gap-8 px-8 pt-3 max-h-[600px] overflow-y-auto bg-grayA-2">
        {conditions.map((cond, index) => (
          <Fragment key={cond.id}>
            <MatchConditionCard
              condition={cond}
              onChange={(updated) =>
                onChange(conditions.map((c) => (c.id === updated.id ? updated : c)))
              }
              onDelete={(id) => {
                const next = conditions.filter((c) => c.id !== id);
                onChange(next);
                if (next.length === 0) {
                  setExpanded(false);
                }
              }}
            />
            {index < conditions.length - 1 && <Separator className="bg-gray-2" />}
          </Fragment>
        ))}
      </div>

      <div className="flex py-6 px-8 bg-grayA-2">
        <Button
          type="button"
          variant="outline"
          size="md"
          className="font-medium"
          onClick={() =>
            onChange([
              ...conditions,
              { id: crypto.randomUUID(), type: "path", mode: "exact", value: "" },
            ])
          }
        >
          <Plus iconSize="sm-regular" />
          Add Condition
        </Button>
      </div>
    </div>
  );
}
