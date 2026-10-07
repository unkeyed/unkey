"use client";

import { Switch } from "@/components/ui/switch";
import { POLICY_LIMITS } from "@/lib/collections/deploy/policies.schema";
import { FormDescription, FormInput } from "@unkey/ui";
import { useFieldArray, useFormContext, useFormState, useWatch } from "react-hook-form";
import { FieldSection, RemoveRowButton, parseOptionalInt } from "../field-parts";
import type { KeyauthFormValues, KeyauthRatelimitFormValues } from "./model";

export function KeyRatelimitsField() {
  const { control } = useFormContext<KeyauthFormValues>();
  const { fields, append, remove } = useFieldArray({ control, name: "ratelimits" });

  return (
    <FieldSection
      label="Key Rate Limits"
      htmlFor="keyauth-ratelimits"
      tooltip="Enforces rate limits defined on the key, by name."
      onAdd={
        fields.length < POLICY_LIMITS.maxRatelimitsPerKeyauth
          ? () => append({ id: crypto.randomUUID(), name: "", override: false })
          : null
      }
    >
      {fields.map((field, index) => (
        <KeyRatelimitRow key={field.id} index={index} onRemove={() => remove(index)} />
      ))}
      {fields.length > 0 ? (
        <FormDescription
          descriptionId="keyauth-ratelimits-desc"
          errorId="keyauth-ratelimits-error"
          description="Override replaces the key's limit, duration and cost for this route."
        />
      ) : null}
    </FieldSection>
  );
}

function KeyRatelimitRow({ index, onRemove }: { index: number; onRemove: () => void }) {
  const { control, setValue } = useFormContext<KeyauthFormValues>();
  const row = useWatch({ control, name: `ratelimits.${index}` });
  const { errors, isSubmitted } = useFormState({ control, name: `ratelimits.${index}` });
  const rowErrors = errors.ratelimits?.[index];
  const message =
    rowErrors?.name?.message ??
    rowErrors?.limit?.message ??
    rowErrors?.duration?.message ??
    rowErrors?.cost?.message;

  const update = (next: KeyauthRatelimitFormValues) =>
    setValue(`ratelimits.${index}`, next, { shouldDirty: true, shouldValidate: isSubmitted });

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center gap-2">
        <FormInput
          placeholder="name (e.g. expensive)"
          value={row.name}
          onChange={(e) => update({ ...row, name: e.target.value })}
          className="flex-1"
          variant={rowErrors?.name ? "error" : undefined}
          aria-invalid={Boolean(rowErrors?.name)}
        />
        <div className="flex items-center gap-1.5 shrink-0 text-xs text-gray-9">
          <Switch
            size="sm"
            checked={row.override}
            // Toggling drops the inline values, so a disabled override never
            // leaks onto the wire.
            onCheckedChange={(override) => update({ id: row.id, name: row.name, override })}
            aria-label="Override limit"
          />
          Override
        </div>
        <RemoveRowButton label="Remove rate limit" onClick={onRemove} />
      </div>
      {row.override ? (
        <div className="flex items-center gap-2 pl-1">
          <OverrideInput
            placeholder="limit"
            value={row.limit}
            invalid={Boolean(rowErrors?.limit)}
            className="flex-1"
            onChange={(limit) => update({ ...row, limit })}
          />
          <OverrideInput
            placeholder="duration (ms)"
            value={row.duration}
            invalid={Boolean(rowErrors?.duration)}
            className="flex-1"
            onChange={(duration) => update({ ...row, duration })}
          />
          <OverrideInput
            placeholder="cost"
            value={row.cost}
            invalid={Boolean(rowErrors?.cost)}
            className="w-20 shrink-0"
            onChange={(cost) => update({ ...row, cost })}
          />
        </div>
      ) : null}
      {message ? (
        <FormDescription
          error={message}
          descriptionId={`ratelimit-${row.id}-desc`}
          errorId={`ratelimit-${row.id}-error`}
        />
      ) : null}
    </div>
  );
}

function OverrideInput({
  placeholder,
  value,
  invalid,
  className,
  onChange,
}: {
  placeholder: string;
  value: number | undefined;
  invalid: boolean;
  className: string;
  onChange: (value: number | undefined) => void;
}) {
  return (
    <FormInput
      type="number"
      placeholder={placeholder}
      value={value ?? ""}
      onChange={(e) => onChange(parseOptionalInt(e.target.value))}
      className={className}
      variant={invalid ? "error" : undefined}
      aria-invalid={invalid}
    />
  );
}
