"use client";

import { cn } from "@/lib/utils";
import { Plus } from "@unkey/icons";
import { Button, Separator } from "@unkey/ui";
import { Fragment } from "react";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import { AccordionSection } from "../accordion-section";
import { summarizeMatchConditions } from "../policy-summaries";
import type { PolicyFormValues } from "../schema";
import { MatchConditionCard } from "./condition-card";
import type { ConditionFieldErrors } from "./condition-fields";

export const MAX_MATCH_CONDITIONS = 10;

export function MatchConditionEditorBody() {
  const { control, setValue } = useFormContext<PolicyFormValues>();
  const { errors, isSubmitted } = useFormState({ control });
  const conditions = useWatch({ control, name: "matchConditions" }) ?? [];
  const conditionErrors = errors.matchConditions as
    | Record<number, ConditionFieldErrors>
    | undefined;
  const onChange = (next: typeof conditions) =>
    setValue("matchConditions", next, {
      shouldDirty: true,
      // Only re-validate after a failed submit so errors clear as the user
      // fixes them, but don't show errors before the first submit attempt.
      shouldValidate: isSubmitted,
    });

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
                errors={conditionErrors?.[index]}
                onChange={(updated) =>
                  onChange(conditions.map((c) => (c.id === updated.id ? updated : c)))
                }
                onDelete={(id) => {
                  onChange(conditions.filter((c) => c.id !== id));
                }}
              />
              {index < conditions.length - 1 && <Separator className="bg-gray-4" />}
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

/**
 * Live-subscribing summary for the active fold's accordion header. Reads only
 * `matchConditions` from the surrounding `FormProvider`.
 */
export function MatchConditionsSummary() {
  const { control } = useFormContext<PolicyFormValues>();
  const conditions = useWatch({ control, name: "matchConditions" });
  return <>{summarizeMatchConditions(conditions ?? [])}</>;
}

/**
 * The collapsed match-conditions accordion row, including its `Clear all`
 * header action. Reads `matchConditions` from form context so the parent
 * panel doesn't have to drill `control` / `setValue` through.
 */
export function MatchConditionsCollapsedSection({ onToggle }: { onToggle: () => void }) {
  const { control, setValue } = useFormContext<PolicyFormValues>();
  const { isSubmitted } = useFormState({ control });
  const conditions = useWatch({ control, name: "matchConditions" }) ?? [];
  return (
    <AccordionSection
      label="Match Conditions"
      summary={summarizeMatchConditions(conditions)}
      active={false}
      onToggle={onToggle}
      tooltipContent={
        <span>
          All conditions must match (<span className="text-gray-12 font-medium">AND</span> logic).
        </span>
      }
      headerAction={
        conditions.length > 0 ? (
          <button
            type="button"
            onClick={() =>
              setValue("matchConditions", [], {
                shouldDirty: true,
                shouldValidate: isSubmitted,
              })
            }
            className="text-xs text-accent-11 hover:text-accent-12 transition-colors cursor-pointer"
          >
            Clear all
          </button>
        ) : undefined
      }
    >
      {null}
    </AccordionSection>
  );
}
