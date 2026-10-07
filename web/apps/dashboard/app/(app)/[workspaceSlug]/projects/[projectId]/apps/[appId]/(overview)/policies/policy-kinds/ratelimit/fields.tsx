"use client";

import { POLICY_LIMITS } from "@/lib/collections/deploy/policies.schema";
import { parseDuration } from "@/lib/duration";
import { formatMs } from "@/lib/ms";
import { IconChevronDownOutline18 } from "@unkey/icons";
import {
  FormDescription,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import type React from "react";
import { useState } from "react";
import {
  useController,
  useFieldArray,
  useFormContext,
  useFormState,
  useWatch,
} from "react-hook-form";
import { DottedLink } from "../../../../components/dotted-link";
import { FieldSection, RemoveRowButton } from "../field-parts";
import { IDENTIFIER_SOURCES } from "./identifier-sources";
import {
  type RatelimitFormValues,
  type RatelimitIdentifierRowValues,
  rateLimitIdentifierSourceSchema,
  ratelimitSummary,
} from "./model";

const RATE_LIMIT_DOCS_URL = "https://www.unkey.com/docs/compute/gateway/rate-limiting#settings";

const IDENTIFIER_SOURCE_OPTIONS = rateLimitIdentifierSourceSchema.options.map((source) => ({
  value: source,
  label: IDENTIFIER_SOURCES[source].label,
}));

function explainIdentifiers(rows: RatelimitIdentifierRowValues[]): React.ReactNode {
  if (rows.length === 0) {
    return null;
  }
  const phrases = rows.map((row) => {
    const { phrase } = IDENTIFIER_SOURCES[row.source];
    return row.value ? `${phrase} (${row.value})` : phrase;
  });
  const needsAuth = rows.some((row) => IDENTIFIER_SOURCES[row.source].needsIdentity);
  return (
    <>
      {rows.length === 1 ? (
        <>
          The policy counts requests for each{" "}
          <span className="text-gray-12 font-medium">{phrases[0]}</span>.
        </>
      ) : (
        <>
          The policy counts requests for each unique combination of{" "}
          <span className="text-gray-12 font-medium">{phrases.join(" × ")}</span>.
        </>
      )}{" "}
      {needsAuth && (
        <>
          Put a Key Auth policy before this policy in the list.{" "}
          <DottedLink href={RATE_LIMIT_DOCS_URL}>Read more</DottedLink>
        </>
      )}
    </>
  );
}

export function RateLimitFields() {
  const { control, setValue } = useFormContext<RatelimitFormValues>();

  const {
    field: { value: limit, onChange: onLimitChange },
    fieldState: { error: limitError },
  } = useController({ control, name: "limit" });

  const {
    field: { value: windowMs, onChange: onWindowChange },
    fieldState: { error: windowError },
  } = useController({ control, name: "windowMs" });

  const [windowDisplay, setWindowDisplay] = useState(() => formatMs(windowMs));
  const [windowParseError, setWindowParseError] = useState<string>();

  const { fields, append, remove } = useFieldArray({ control, name: "identifiers" });
  const rows = useWatch({ control, name: "identifiers" }) ?? [];
  const { errors, isSubmitted } = useFormState({ control, name: "identifiers" });
  const updateRow = (index: number, row: RatelimitIdentifierRowValues) =>
    setValue(`identifiers.${index}`, row, { shouldDirty: true, shouldValidate: isSubmitted });

  return (
    <div className="flex flex-col gap-4">
      <div className="flex gap-3">
        <FormInput
          label="Limit"
          type="number"
          value={limit}
          onChange={(e) => onLimitChange(Number.parseInt(e.target.value) || 0)}
          className="flex-1"
          error={limitError?.message}
        />
        <FormInput
          label="Window"
          type="text"
          value={windowDisplay}
          placeholder="e.g. 5s, 2m, 1h, 500ms"
          onChange={(e) => {
            const raw = e.target.value;
            setWindowDisplay(raw);

            const trimmed = raw.trim();
            if (trimmed === "") {
              setWindowParseError(undefined);
              onWindowChange(0);
              return;
            }

            const asNumber = Number(trimmed);
            if (Number.isFinite(asNumber) && asNumber > 0) {
              setWindowParseError(undefined);
              onWindowChange(Math.floor(asNumber));
              return;
            }

            const parsed = parseDuration(trimmed);
            if (parsed > 0) {
              setWindowParseError(undefined);
              onWindowChange(parsed);
            } else {
              setWindowParseError('Use a duration like "5s", "2m", "1h" or milliseconds');
            }
          }}
          className="flex-1"
          descriptionPosition="label"
          description={
            windowMs > 0
              ? `The counter resets every ${formatMs(windowMs, { long: true })}.`
              : undefined
          }
          error={windowParseError ?? windowError?.message}
        />
      </div>

      <FieldSection
        label="Identifiers"
        htmlFor="ratelimit-identifiers"
        addDisabled={fields.length >= POLICY_LIMITS.maxIdentifiersPerRatelimit}
        onAdd={() => append({ id: crypto.randomUUID(), source: "path", value: "" })}
      >
        {fields.map((field, index) => {
          const row = rows[index] ?? field;
          const rowError = errors.identifiers?.[index]?.value?.message;
          const { valuePlaceholder } = IDENTIFIER_SOURCES[row.source];
          return (
            <div key={field.id} className="flex flex-col gap-1">
              <div className="flex items-center gap-2">
                <div className="w-48 shrink-0">
                  <Select
                    value={row.source}
                    items={IDENTIFIER_SOURCE_OPTIONS}
                    onValueChange={(v) => {
                      const source = rateLimitIdentifierSourceSchema.options.find((s) => s === v);
                      if (source) {
                        updateRow(index, { ...row, source, value: "" });
                      }
                    }}
                  >
                    <SelectTrigger
                      aria-label="Identifier source"
                      className="shrink-0 whitespace-pre"
                      rightIcon={<IconChevronDownOutline18 className="size-3.5 absolute right-2" />}
                    >
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {IDENTIFIER_SOURCE_OPTIONS.map((opt) => (
                        <SelectItem
                          key={opt.value}
                          value={opt.value}
                          className="shrink-0 whitespace-pre"
                        >
                          {opt.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                {valuePlaceholder !== null ? (
                  <FormInput
                    placeholder={valuePlaceholder}
                    value={row.value}
                    onChange={(e) => updateRow(index, { ...row, value: e.target.value })}
                    className="flex-1"
                    variant={rowError ? "error" : undefined}
                    aria-invalid={Boolean(rowError)}
                  />
                ) : (
                  <span className="flex-1" />
                )}
                <RemoveRowButton
                  label="Remove identifier"
                  disabled={fields.length === 1}
                  onClick={() => remove(index)}
                />
              </div>
              {rowError && (
                <FormDescription
                  error={rowError}
                  descriptionId={`identifier-${index}-desc`}
                  errorId={`identifier-${index}-error`}
                />
              )}
            </div>
          );
        })}

        <p className="text-gray-11 text-xs leading-5">{explainIdentifiers(rows)}</p>
      </FieldSection>
    </div>
  );
}

export function RatelimitSummary() {
  const { control } = useFormContext<RatelimitFormValues>();
  const [limit, windowMs, identifiers] = useWatch({
    control,
    name: ["limit", "windowMs", "identifiers"],
  });
  return ratelimitSummary({ limit, windowMs, identifiers });
}
