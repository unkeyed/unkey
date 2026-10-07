"use client";

import { httpMethodSchema } from "@/lib/collections/deploy/policies.schema";
import { IconXmarkOutline12 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Input } from "@unkey/ui";
import { cn } from "cn";
import { useState } from "react";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import type { PolicyFormValues } from "../../../policy-kinds";
import { CompactMultiSelect, CompactSelect } from "./compact-select";
import {
  CONDITION_TYPES,
  type ConditionNameField,
  type ConditionRegex,
  type ConditionValueField,
  type OperatorControl,
  conditionRowState,
  getDefaultCondition,
  withOperator,
} from "./condition-kinds";
import type { ConditionOperator, MatchConditionFormValues } from "./condition-schema";
import { RegexGenerate } from "./regex-generate";
import { ROW, RuleLabel } from "./rule-row";

const METHOD_OPTIONS = httpMethodSchema.options.map((m) => ({ value: m, label: m }));
const INPUT = "h-8 min-w-0 flex-1 font-mono text-xs";

function ConditionValue({
  condition,
  field,
  invalid,
  onChange,
}: {
  condition: MatchConditionFormValues;
  field: ConditionValueField;
  invalid: boolean;
  onChange: (next: MatchConditionFormValues) => void;
}) {
  if (field.type === "none") {
    return <span className="min-w-0 flex-1" />;
  }
  if (condition.type === "method") {
    return (
      <CompactMultiSelect
        label={field.ariaLabel}
        className="min-w-0 flex-1"
        values={condition.methods}
        options={METHOD_OPTIONS}
        onChange={(methods) => onChange({ ...condition, methods })}
        placeholder={field.placeholder}
        invalid={invalid}
      />
    );
  }
  return (
    <Input
      aria-label={field.ariaLabel}
      aria-invalid={invalid}
      className={INPUT}
      placeholder={field.placeholder}
      value={condition.value}
      onChange={(e) => onChange({ ...condition, value: e.target.value })}
    />
  );
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
      className={INPUT}
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
      <CompactSelect
        label="Operator"
        className={cn(width, "shrink-0")}
        value={value}
        options={options}
        onChange={onChange}
      />
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
  // A row added or retyped after a failed submit stays quiet until the next submit.
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
        condition={condition}
        field={state.value}
        invalid={!nameError && Boolean(valueError)}
        onChange={patch}
      />
    </>
  );

  return (
    <div className={ROW}>
      <div className="flex h-12 items-center gap-2 pr-2 pl-3">
        <RuleLabel>{label}</RuleLabel>
        <CompactSelect
          label="Condition field"
          className="w-32 shrink-0"
          value={condition.type}
          options={CONDITION_TYPES}
          onChange={(type) => {
            setErrorsHiddenThroughSubmit(formState.submitCount);
            patch(getDefaultCondition(type, condition.id));
          }}
        />
        {state.name.type === "name" ? (
          <ConditionName field={state.name} error={nameError} onChange={patch} />
        ) : (
          operatorAndValue
        )}
        <button
          type="button"
          aria-label="Remove condition"
          onClick={onRemove}
          className="flex size-8 shrink-0 items-center justify-center rounded-md text-gray-10 transition-colors hover:bg-grayA-3 hover:text-gray-12"
        >
          <IconXmarkOutline12 className="size-3" />
        </button>
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
    </div>
  );
}
