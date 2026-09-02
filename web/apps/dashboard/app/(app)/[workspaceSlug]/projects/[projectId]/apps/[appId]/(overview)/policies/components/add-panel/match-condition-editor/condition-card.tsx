"use client";

import { Button, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { IconChevronDownOutline18, IconTrashOutline18 } from "nucleo-ui-outline-18";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import {
  getDefaultCondition,
  type MatchConditionFormValues,
  type PolicyFormValues,
} from "../schema";
import { ConditionFields } from "./condition-fields";
import { MATCH_TYPE_OPTIONS } from "./constants";

export function MatchConditionCard({ index, onRemove }: { index: number; onRemove: () => void }) {
  const { control, setValue } = useFormContext<PolicyFormValues>();
  const allConditions = useWatch({ control, name: "matchConditions" });
  const condition = allConditions?.[index];
  const { errors } = useFormState({ control, name: `matchConditions.${index}` });

  const conditionErrors = (
    errors.matchConditions as Record<number, Record<string, { message?: string }>> | undefined
  )?.[index];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-4">
        <div className="flex-1">
          <Select
            value={condition?.type}
            items={MATCH_TYPE_OPTIONS}
            onValueChange={(v) => {
              const newType = v as MatchConditionFormValues["type"];
              setValue(`matchConditions.${index}`, getDefaultCondition(newType, condition?.id));
            }}
          >
            <SelectTrigger
              aria-label="Condition type"
              rightIcon={<IconChevronDownOutline18 className="size-4 absolute right-2" />}
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
          <IconTrashOutline18 />
        </Button>
      </div>
      <ConditionFields index={index} errors={conditionErrors} />
    </div>
  );
}
