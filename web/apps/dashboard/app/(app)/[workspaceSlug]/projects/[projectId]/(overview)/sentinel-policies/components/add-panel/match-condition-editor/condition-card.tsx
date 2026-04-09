import type { StringMatchMode } from "@/lib/collections/deploy/sentinel-policies.schema";
import { ChevronDown, Trash } from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import type { MatchConditionFormValues } from "../schema";
import { type ConditionFieldErrors, ConditionFields } from "./condition-fields";
import { type HttpMethod, MATCH_TYPE_OPTIONS } from "./constants";

export function MatchConditionCard({
  condition,
  errors,
  onChange,
  onDelete,
}: {
  condition: MatchConditionFormValues;
  errors?: ConditionFieldErrors;
  onChange: (updated: MatchConditionFormValues) => void;
  onDelete: (id: string) => void;
}) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-4">
        <div className="flex-1">
          <Select
            value={condition.type}
            onValueChange={(v) => {
              const type = v as MatchConditionFormValues["type"];
              const base = { id: condition.id };
              const reset: MatchConditionFormValues = match(type)
                .with("path", () => ({
                  ...base,
                  type: "path" as const,
                  mode: "exact" as StringMatchMode,
                  value: "",
                }))
                .with("method", () => ({
                  ...base,
                  type: "method" as const,
                  methods: [] as HttpMethod[],
                }))
                .with("header", () => ({ ...base, type: "header" as const, name: "" }))
                .with("queryParam", () => ({ ...base, type: "queryParam" as const, name: "" }))
                .exhaustive();
              onChange(reset);
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
          onClick={() => onDelete(condition.id)}
        >
          <Trash iconSize="sm-regular" />
        </Button>
      </div>
      <ConditionFields condition={condition} onChange={onChange} errors={errors} />
    </div>
  );
}
