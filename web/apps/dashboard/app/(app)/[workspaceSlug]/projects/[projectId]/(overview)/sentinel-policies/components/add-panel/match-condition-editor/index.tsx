"use client";

import { cn } from "@/lib/utils";
import { Plus } from "@unkey/icons";
import { Button, Separator } from "@unkey/ui";
import { Fragment } from "react";
import { useFieldArray, useFormContext, useFormState, useWatch } from "react-hook-form";
import { AccordionSection } from "../accordion-section";
import { summarizeMatchConditions } from "../policy-summaries";
import { type PolicyFormValues, getDefaultCondition } from "../schema";
import { type ConditionFieldErrors, MatchConditionCard } from "./condition-card";
import { SENTINEL_LIMITS } from "@/lib/collections/deploy/sentinel-policies.schema";


export function MatchConditionEditorBody() {
  const { control } = useFormContext<PolicyFormValues>();
  const { errors } = useFormState({ control });
  const { fields, append, remove, update } = useFieldArray({
    control,
    name: "matchConditions",
  });

  const conditionErrors = errors.matchConditions as
    | Record<number, ConditionFieldErrors>
    | undefined;

  const atCap = fields.length >= SENTINEL_LIMITS.maxMatchExprsPerPolicy;
  const hasConditions = fields.length > 0;
  return (
    <div className="flex flex-col">
      {hasConditions && (
        <div className="flex flex-col gap-8 pt-3">
          {fields.map((field, index) => (
            <Fragment key={field.id}>
              <MatchConditionCard
                condition={field}
                errors={conditionErrors?.[index]}
                onChange={(updated) => update(index, updated)}
                onRemove={() => remove(index)}
              />
              {index < fields.length - 1 && <Separator className="bg-gray-4" />}
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
          onClick={() => append(getDefaultCondition("path"))}
        >
          <Plus iconSize="sm-regular" />
          {fields.length === 0 ? "Add First Condition" : "Add Condition"}
        </Button>
        <span className="text-[12px] text-gray-11">
          {fields.length} / {SENTINEL_LIMITS.maxMatchExprsPerPolicy}
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
