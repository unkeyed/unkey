"use client";

import { POLICY_LIMITS } from "@/lib/collections/deploy/policies.schema";
import { IconChevronDownOutline18, IconPlusOutline12 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "cn";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { POLICY_KINDS, type PolicyFormValues } from "../../../policy-kinds";
import { getDefaultCondition } from "../../../policy-kinds/conditions/kinds";
import { KindSummary, POLICY_KIND_UI } from "../../../policy-kinds/fields";
import { RejectionTags } from "../../rejections";
import { ConditionRow } from "./condition-row";
import { RuleLabel, RuleRow } from "./rule-row";

export function RuleCard({
  configOpen,
  onConfigOpenChange,
}: {
  configOpen: boolean;
  onConfigOpenChange: (open: boolean) => void;
}) {
  const { control } = useFormContext<PolicyFormValues>();
  const type = useWatch({ control, name: "type" });
  const { fields, append, remove } = useFieldArray({ control, name: "matchConditions" });
  const { Icon, label, rejects } = POLICY_KINDS[type];
  const { Fields } = POLICY_KIND_UI[type];
  const OpenFields = configOpen ? Fields : null;
  const atCap = fields.length >= POLICY_LIMITS.maxMatchExprsPerPolicy;

  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-sm font-medium text-gray-12">Rule</span>
      <div className="flex flex-col gap-1.5 rounded-lg bg-grayA-2 p-2">
        {fields.length === 0 ? (
          <RuleRow className="flex h-11 items-center gap-2 px-3">
            <RuleLabel>If</RuleLabel>
            <span className="text-sm text-gray-11">Every request</span>
          </RuleRow>
        ) : (
          fields.map((field, index) => (
            <ConditionRow
              key={field.id}
              index={index}
              label={index === 0 ? "If" : "And"}
              onRemove={() => remove(index)}
            />
          ))
        )}
        <div className="pl-14">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={atCap}
            title={
              atCap
                ? `A policy holds at most ${POLICY_LIMITS.maxMatchExprsPerPolicy} conditions.`
                : undefined
            }
            onClick={() => append(getDefaultCondition("path"))}
            className="text-xs text-gray-11 hover:text-gray-12"
          >
            <IconPlusOutline12 className="size-3" />
            Add condition
          </Button>
        </div>
        <RuleRow>
          <button
            type="button"
            aria-expanded={Fields ? OpenFields !== null : undefined}
            disabled={!Fields}
            onClick={() => onConfigOpenChange(!configOpen)}
            className="flex h-11 w-full items-center gap-2 rounded-md px-3 text-left transition-colors enabled:hover:bg-grayA-2"
          >
            <RuleLabel>Then</RuleLabel>
            <Icon className="size-3.5 shrink-0 text-gray-11" />
            <span className="shrink-0 text-sm font-medium text-gray-12">{label}</span>
            <span className="min-w-0 flex-1 truncate text-xs text-gray-11">
              · <KindSummary type={type} />
            </span>
            {Fields ? (
              <IconChevronDownOutline18
                className={cn(
                  "size-3.5 shrink-0 text-gray-11 transition-transform duration-150 ease-out motion-reduce:transition-none",
                  OpenFields && "rotate-180",
                )}
              />
            ) : null}
          </button>
          {OpenFields ? (
            <div className="border-t border-grayA-4 px-4 pt-4 pb-5">
              <OpenFields />
            </div>
          ) : null}
          {OpenFields && rejects.length > 0 ? (
            <div className="flex items-start gap-2 border-t border-grayA-4 px-3 py-3">
              <span className="flex h-5 items-center">
                <RuleLabel>On fail</RuleLabel>
              </span>
              <RejectionTags rejects={rejects} />
            </div>
          ) : null}
        </RuleRow>
      </div>
    </div>
  );
}
