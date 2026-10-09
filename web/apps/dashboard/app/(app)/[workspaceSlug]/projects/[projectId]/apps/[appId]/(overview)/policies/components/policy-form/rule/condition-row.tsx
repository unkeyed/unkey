"use client";

import { httpMethodSchema } from "@/lib/collections/deploy/policies.schema";
import { match } from "@unkey/match";
import { Input, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { cn } from "cn";
import { useState } from "react";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import type { PolicyFormValues } from "../../../policy-kinds";
import {
  CONDITION_TYPES,
  type ConditionNameField,
  type ConditionRegex,
  type ConditionValueField,
  type OperatorControl,
  conditionRowState,
  getDefaultCondition,
  withOperator,
} from "../../../policy-kinds/conditions/kinds";
import type {
  ConditionOperator,
  MatchConditionFormValues,
} from "../../../policy-kinds/conditions/schema";
import { RemoveRowButton } from "../../../policy-kinds/field-parts";
import { RegexGenerate } from "./regex-generate";
import { RuleLabel, RuleRow } from "./rule-row";

const METHOD_OPTIONS = httpMethodSchema.options.map((m) => ({ value: m, label: m }));

function ConditionValue({
  field,
  invalid,
  onChange,
}: {
  field: ConditionValueField;
  invalid: boolean;
  onChange: (next: MatchConditionFormValues) => void;
}) {
  return match(field)
    .with({ type: "none" }, () => <span className="min-w-0 flex-1" />)
    .with({ type: "methods" }, ({ condition, ariaLabel, placeholder }) => (
      <Select
        multiple
        value={condition.methods}
        items={METHOD_OPTIONS}
        onValueChange={(methods) => onChange({ ...condition, methods })}
      >
        <SelectTrigger
          size="sm"
          aria-label={ariaLabel}
          aria-invalid={invalid}
          wrapperClassName="min-w-0 flex-1"
          className="font-mono text-xs"
        >
          <SelectValue
            placeholder={placeholder}
            className="block truncate data-placeholder:font-sans data-placeholder:text-grayA-8"
          />
        </SelectTrigger>
        <SelectContent>
          {METHOD_OPTIONS.map((o) => (
            <SelectItem key={o.value} value={o.value} className="font-mono text-xs">
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    ))
    .with({ type: "text" }, ({ condition, ariaLabel, placeholder }) => (
      <Input
        aria-label={ariaLabel}
        aria-invalid={invalid}
        className="h-8 min-w-0 flex-1 font-mono text-xs"
        placeholder={placeholder}
        value={condition.value}
        onChange={(e) => onChange({ ...condition, value: e.target.value })}
      />
    ))
    .exhaustive();
}

function ConditionName({
  field: { label, placeholder, condition },
  error,
  onChange,
}: {
  field: Extract<ConditionNameField, { type: "name" }>;
  error: string | undefined;
  onChange: (next: MatchConditionFormValues) => void;
}) {
  return (
    <Input
      aria-label={label}
      aria-invalid={Boolean(error)}
      className="h-8 min-w-0 flex-1 font-mono text-xs"
      placeholder={placeholder}
      value={condition.name}
      onChange={(e) => onChange({ ...condition, name: e.target.value })}
    />
  );
}

function OperatorField({
  control,
  width,
  onChange,
}: {
  control: OperatorControl;
  width: string;
  onChange: (op: ConditionOperator) => void;
}) {
  return match(control)
    .with({ type: "fixed" }, ({ label }) => (
      <span className={cn(width, "shrink-0 px-1 text-sm text-gray-11")}>{label}</span>
    ))
    .with({ type: "select" }, ({ value, options }) => (
      <Select
        value={value}
        items={options}
        onValueChange={(op) => {
          if (op !== null) {
            onChange(op);
          }
        }}
      >
        <SelectTrigger size="sm" aria-label="Operator" wrapperClassName={cn(width, "shrink-0")}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    ))
    .exhaustive();
}

function ConditionRegexGenerate({
  regex,
  onChange,
}: {
  regex: ConditionRegex;
  onChange: (next: MatchConditionFormValues) => void;
}) {
  return match(regex)
    .with({ type: "none" }, () => null)
    .with({ type: "regex" }, ({ condition, prompt }) => (
      <div className="border-t border-grayA-4 py-2.5 pr-2 pl-17">
        <RegexGenerate
          conditionType={condition.type}
          placeholder={prompt}
          onGenerated={(pattern) => onChange({ ...condition, value: pattern })}
        />
      </div>
    ))
    .exhaustive();
}

export function ConditionRow({
  index,
  label,
  onRemove,
}: {
  index: number;
  label: string;
  onRemove: () => void;
}) {
  const { control, setValue, getFieldState } = useFormContext<PolicyFormValues>();
  const condition = useWatch({ control, name: "matchConditions" })?.[index];
  const formState = useFormState({ control, name: "matchConditions" });
  const [errorsHiddenThroughSubmit, setErrorsHiddenThroughSubmit] = useState(formState.submitCount);

  if (!condition) {
    return null;
  }

  const fieldError = (field: "name" | "value" | "methods") =>
    formState.submitCount > errorsHiddenThroughSubmit
      ? getFieldState(`matchConditions.${index}.${field}`, formState).error?.message
      : undefined;

  const patch = (next: MatchConditionFormValues) => {
    setValue(`matchConditions.${index}`, next, { shouldValidate: formState.isSubmitted });
  };

  const state = conditionRowState(condition);
  const nameError = fieldError("name");
  const valueError =
    (state.regex.type === "regex" ? state.regex.syntaxError : undefined) ??
    fieldError("value") ??
    fieldError("methods");

  const operatorAndValue = (
    <>
      <OperatorField
        control={state.operator}
        width={state.name.type === "name" ? "w-32" : "w-36"}
        onChange={(op) => patch(withOperator(condition, op))}
      />
      <ConditionValue
        field={state.value}
        invalid={!nameError && Boolean(valueError)}
        onChange={patch}
      />
    </>
  );

  return (
    <RuleRow>
      <div className="flex h-12 items-center gap-2 pr-2 pl-3">
        <RuleLabel>{label}</RuleLabel>
        <Select
          value={condition.type}
          items={CONDITION_TYPES}
          onValueChange={(type) => {
            if (type !== null) {
              setErrorsHiddenThroughSubmit(formState.submitCount);
              patch(getDefaultCondition(type));
            }
          }}
        >
          <SelectTrigger size="sm" aria-label="Condition field" wrapperClassName="w-32 shrink-0">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {CONDITION_TYPES.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {state.name.type === "name" ? (
          <ConditionName field={state.name} error={nameError} onChange={patch} />
        ) : (
          operatorAndValue
        )}
        <RemoveRowButton label="Remove condition" onClick={onRemove} />
      </div>
      {state.name.type === "name" ? (
        <div className="-mt-1 flex h-11 items-center gap-2 pr-12 pb-1 pl-17">
          {operatorAndValue}
        </div>
      ) : null}
      {nameError || valueError ? (
        <p className="-mt-1 pr-3 pb-2.5 pl-17 text-xs text-error-11">{nameError ?? valueError}</p>
      ) : null}
      <ConditionRegexGenerate regex={state.regex} onChange={patch} />
    </RuleRow>
  );
}
