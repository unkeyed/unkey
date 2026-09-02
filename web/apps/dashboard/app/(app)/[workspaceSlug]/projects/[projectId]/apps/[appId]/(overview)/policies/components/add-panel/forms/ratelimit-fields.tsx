"use client";

import {
  Button,
  FormDescription,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { FormLabel } from "@unkey/ui/src/components/form/form-helpers";
import {
  IconChevronDownOutline18,
  IconPlusOutline18,
  IconTrashOutline18,
} from "nucleo-ui-outline-18";
import type React from "react";
import { useState } from "react";
import {
  useController,
  useFieldArray,
  useFormContext,
  useFormState,
  useWatch,
} from "react-hook-form";
import { POLICY_LIMITS } from "@/lib/collections/deploy/policies.schema";
import { parseDuration } from "@/lib/duration";
import { formatMs } from "@/lib/ms";
import type { RateLimitIdentifierSource, RatelimitIdentifierRowValues } from "../schema";
import { DocsLink, Sep, Strong } from "./summary-helpers";

type RatelimitFormValues = {
  type: "ratelimit";
  name: string;
  environmentId: string;
  limit: number;
  windowMs: number;
  identifiers: RatelimitIdentifierRowValues[];
};

const IDENTIFIER_SOURCE_LABELS: Record<RateLimitIdentifierSource, string> = {
  remoteIp: "IP",
  header: "Header",
  authenticatedSubject: "Subject",
  path: "Path",
  principalField: "Field",
};

const RATE_LIMIT_DOCS_URL =
  "https://www.unkey.com/docs/platform/gateway/policies/rate-limiting#identifiers";

const IDENTIFIER_SOURCE_OPTIONS: { value: RateLimitIdentifierSource; label: string }[] = [
  { value: "remoteIp", label: "Remote IP" },
  { value: "header", label: "Header" },
  { value: "authenticatedSubject", label: "Authenticated Subject" },
  { value: "path", label: "Request Path" },
  { value: "principalField", label: "Principal Field" },
];

// Human phrase per source for the live explainer sentence below the list.
const IDENTIFIER_SOURCE_PHRASES: Record<RateLimitIdentifierSource, string> = {
  remoteIp: "client IP",
  header: "header value",
  authenticatedSubject: "authenticated subject",
  path: "request path",
  principalField: "principal field value",
};

function needsValue(source: RateLimitIdentifierSource): boolean {
  return source === "header" || source === "principalField";
}

function explainIdentifiers(rows: RatelimitIdentifierRowValues[]): React.ReactNode {
  if (rows.length === 0) {
    return null;
  }
  const phrases = rows.map((row) => {
    const phrase = IDENTIFIER_SOURCE_PHRASES[row.source];
    return row.value ? `${phrase} (${row.value})` : phrase;
  });
  const needsAuth = rows.some(
    (row) => row.source === "authenticatedSubject" || row.source === "principalField",
  );
  return (
    <>
      {rows.length === 1 ? (
        <>
          The policy counts requests for each <Strong>{phrases[0]}</Strong>.
        </>
      ) : (
        <>
          The policy counts requests for each unique combination of{" "}
          <Strong>{phrases.join(" × ")}</Strong>. Each combination has its own counter.
        </>
      )}{" "}
      {needsAuth && (
        <>
          Put a Key Auth or JWT Auth policy before this policy in the list.{" "}
          <DocsLink href={RATE_LIMIT_DOCS_URL} />
        </>
      )}
    </>
  );
}

export function RateLimitFields() {
  const { control } = useFormContext<RatelimitFormValues>();

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

  const { fields, append, remove, update } = useFieldArray({ control, name: "identifiers" });
  const rows = useWatch({ control, name: "identifiers" }) ?? [];
  const { errors } = useFormState({ control, name: "identifiers" });

  return (
    <div className="flex flex-col gap-4">
      <div className="flex gap-3">
        <FormInput
          label="Limit"
          descriptionPosition="label"
          description="Maximum number of requests in one window."
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
              : "The time until the counter resets."
          }
          error={windowParseError ?? windowError?.message}
        />
      </div>

      <fieldset className="flex flex-col gap-2 border-0 m-0 p-0">
        <div className="flex items-center justify-between">
          <FormLabel
            label="Identifiers"
            htmlFor="ratelimit-identifiers"
            tooltipContent="The policy counts requests for each unique combination of the resolved values. Example: [Subject, Path] gives each subject its own limit on each path."
          />
          <Button
            type="button"
            variant="outline"
            size="md"
            className="font-medium"
            disabled={fields.length >= POLICY_LIMITS.maxIdentifiersPerRatelimit}
            onClick={() => append({ id: crypto.randomUUID(), source: "path", value: "" })}
          >
            <IconPlusOutline18 />
            Add
          </Button>
        </div>

        {fields.map((field, index) => {
          const row = rows[index] ?? field;
          const rowError = errors.identifiers?.[index]?.value?.message;
          return (
            <div key={field.id} className="flex flex-col gap-1">
              <div className="flex items-center gap-2">
                <div className="w-48 shrink-0">
                  <Select
                    value={row.source}
                    items={IDENTIFIER_SOURCE_OPTIONS}
                    onValueChange={(v) =>
                      update(index, {
                        ...row,
                        source: v as RateLimitIdentifierSource,
                        value: "",
                      })
                    }
                  >
                    <SelectTrigger
                      aria-label="Identifier source"
                      className="shrink-0 whitespace-pre"
                      rightIcon={<IconChevronDownOutline18 className="size-4 absolute right-2" />}
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
                {needsValue(row.source) ? (
                  <FormInput
                    placeholder={row.source === "header" ? "X-Tenant-Id" : "subject"}
                    requirement="required"
                    value={row.value}
                    onChange={(e) => update(index, { ...row, value: e.target.value })}
                    className="flex-1"
                    variant={rowError ? "error" : undefined}
                    aria-invalid={Boolean(rowError)}
                  />
                ) : (
                  <span className="flex-1" />
                )}
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  aria-label="Remove identifier"
                  className="size-9 shrink-0 px-0 justify-center text-gray-11 hover:text-gray-12 hover:bg-grayA-3 rounded-lg"
                  disabled={fields.length === 1}
                  onClick={() => remove(index)}
                >
                  <IconTrashOutline18 />
                </Button>
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
      </fieldset>
    </div>
  );
}

export function RatelimitPolicySummary() {
  const { control } = useFormContext<RatelimitFormValues>();
  const limit = useWatch({ control, name: "limit" });
  const windowMs = useWatch({ control, name: "windowMs" });
  const identifiers = useWatch({ control, name: "identifiers" }) ?? [];

  return (
    <div className="max-w-75 truncate">
      <span className="text-gray-11">
        <Strong>{limit}</Strong> / {formatMs(windowMs)}
        <Sep />
        per{" "}
        <Strong>
          {identifiers
            .map((row) =>
              row.value
                ? `${IDENTIFIER_SOURCE_LABELS[row.source]}: ${row.value}`
                : IDENTIFIER_SOURCE_LABELS[row.source],
            )
            .join(" × ")}
        </Strong>
      </span>
    </div>
  );
}
