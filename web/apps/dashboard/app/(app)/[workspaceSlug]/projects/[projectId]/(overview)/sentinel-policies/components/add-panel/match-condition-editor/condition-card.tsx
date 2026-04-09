import { ChevronDown, Trash } from "@unkey/icons";
import { Button, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import type { FieldError } from "react-hook-form";
import { type MatchConditionFormValues, getDefaultCondition } from "../schema";
import { ConditionFields } from "./condition-fields";
import { MATCH_TYPE_OPTIONS } from "./constants";

export type ConditionFieldErrors = Partial<Record<string, FieldError>> | undefined;

export function MatchConditionCard({
  condition,
  errors,
  onChange,
  onRemove,
}: {
  condition: MatchConditionFormValues;
  errors?: ConditionFieldErrors;
  onChange: (updated: MatchConditionFormValues) => void;
  onRemove: () => void;
}) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-4">
        <div className="flex-1">
          <Select
            value={condition.type}
            onValueChange={(v) => {
              const type = v as MatchConditionFormValues["type"];
              onChange(getDefaultCondition(type, condition.id));
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
      <ConditionFields condition={condition} onChange={onChange} errors={errors} />
    </div>
  );
}
