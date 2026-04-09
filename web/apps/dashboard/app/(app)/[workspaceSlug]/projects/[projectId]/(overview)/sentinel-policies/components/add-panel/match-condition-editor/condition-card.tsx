"use client";

import { ChevronDown, Trash } from "@unkey/icons";
import { Button, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import {
  type MatchConditionFormValues,
  type PolicyFormValues,
  getDefaultCondition,
} from "../schema";
import { ConditionFields } from "./condition-fields";
import { MATCH_TYPE_OPTIONS } from "./constants";

export function MatchConditionCard({
  index,
  onRemove,
}: {
  index: number;
  onRemove: () => void;
}) {
  const { control, setValue } = useFormContext<PolicyFormValues>();
  // Watch only the fields the card header needs. ConditionFields has its own
  // scoped watch for the rest, so typing in a field input won't re-render
  // the type selector or remove button.
  const type = useWatch({ control, name: `matchConditions.${index}.type` });
  const id = useWatch({ control, name: `matchConditions.${index}.id` });
  const { errors } = useFormState({ control, name: `matchConditions.${index}` });

  const conditionErrors = (
    errors.matchConditions as Record<number, Record<string, { message?: string }>> | undefined
  )?.[index];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-4">
        <div className="flex-1">
          <Select
            value={type}
            onValueChange={(v) => {
              const newType = v as MatchConditionFormValues["type"];
              setValue(`matchConditions.${index}`, getDefaultCondition(newType, id));
            }}
          >
            <SelectTrigger
              aria-label="Condition type"
              rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="z-60">
              {MATCH_TYPE_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          aria-label="Remove condition"
          className="size-9 shrink-0 px-0 justify-center text-gray-11 hover:text-gray-12 hover:bg-grayA-3 rounded-lg"
          onClick={onRemove}
        >
          <Trash iconSize="sm-regular" />
        </Button>
      </div>
      <ConditionFields index={index} errors={conditionErrors} />
    </div>
  );
}
