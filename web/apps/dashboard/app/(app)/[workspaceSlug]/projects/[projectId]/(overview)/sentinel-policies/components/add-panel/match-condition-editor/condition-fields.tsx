"use client";

import type { StringMatchMode } from "@/lib/collections/deploy/sentinel-policies.schema";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { ChevronDown, Sparkle3 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { FormDescription, FormLabel } from "@unkey/ui/src/components/form/form-helpers";
import { useState } from "react";
import type { FieldError } from "react-hook-form";
import type { MatchConditionFormValues } from "../schema";
import { HTTP_METHODS, STRING_MATCH_MODES, validateRegexSyntax } from "./constants";

/** Per-condition field errors. Keyed by field name, values are react-hook-form FieldError. */
export type ConditionFieldErrors = Partial<Record<string, FieldError>> | undefined;

export function ConditionFields({
  condition,
  onChange,
  errors,
}: {
  condition: MatchConditionFormValues;
  onChange: (updated: MatchConditionFormValues) => void;
  errors?: ConditionFieldErrors;
}) {
  return match(condition)
    .with({ type: "path" }, (c) => (
      <div className="flex flex-col gap-4">
        <div className="flex gap-2">
          <div className="w-28 shrink-0">
            <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
              <label htmlFor={`path-mode-${c.id}`} className="text-gray-11 text-[13px]">
                Mode
              </label>
              <Select
                value={c.mode}
                onValueChange={(v) => onChange({ ...c, mode: v as StringMatchMode })}
              >
                <SelectTrigger
                  id={`path-mode-${c.id}`}
                  rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {STRING_MATCH_MODES.map((m) => (
                    <SelectItem key={m.value} value={m.value}>
                      {m.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </fieldset>
          </div>
          <FormInput
            label="Path"
            required
            placeholder={c.mode === "regex" ? "^/api/.*" : "/api/v1"}
            value={c.value}
            onChange={(e) => onChange({ ...c, value: e.target.value })}
            className="flex-1"
            descriptionPosition="label"
            description={
              c.mode === "regex"
                ? "Regex pattern for path matching."
                : "The URL path to match against."
            }
            error={
              (c.mode === "regex" ? validateRegexSyntax(c.value) : undefined) ??
              errors?.value?.message
            }
          />
        </div>
        {c.mode === "regex" && (
          <RegexGenerateInput
            conditionType="path"
            onGenerated={(pattern) => onChange({ ...c, value: pattern })}
          />
        )}
      </div>
    ))
    .with({ type: "method" }, (c) => {
      const methodError = errors?.methods?.message;
      return (
        <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
          <FormLabel
            label="Allowed Methods"
            htmlFor={`method-group-${c.id}`}
            required
            hasError={Boolean(methodError)}
            tooltipContent="Select which HTTP methods this condition matches."
          />
          <div
            id={`method-group-${c.id}`}
            className="flex flex-wrap gap-1.5"
            // biome-ignore lint/a11y/useSemanticElements: its okay
            role="group"
            data-error={methodError ? "true" : undefined}
            tabIndex={methodError ? -1 : undefined}
          >
            {HTTP_METHODS.map((m) => {
              const active = c.methods.includes(m);
              return (
                <button
                  key={m}
                  type="button"
                  aria-pressed={active}
                  onClick={() =>
                    onChange({
                      ...c,
                      methods: active ? c.methods.filter((x) => x !== m) : [...c.methods, m],
                    })
                  }
                  className={cn(
                    "px-2 py-0.5 rounded text-xs font-mono border transition-colors cursor-pointer",
                    active
                      ? "bg-info-3 border-info-7 text-info-11"
                      : "bg-grayA-2 border-grayA-4 text-grayA-9 hover:text-gray-12",
                  )}
                >
                  {m}
                </button>
              );
            })}
          </div>
          <FormDescription
            error={methodError}
            descriptionId={`method-desc-${c.id}`}
            errorId={`method-error-${c.id}`}
          />
        </fieldset>
      );
    })
    .with({ type: "header" }, { type: "queryParam" }, (c) => {
      const isHeader = c.type === "header";
      const conditionType = c.type;
      return (
        <div className="flex flex-col gap-4">
          <FormInput
            label={isHeader ? "Header Name" : "Parameter Name"}
            required
            placeholder={isHeader ? "X-Custom-Header" : "param_name"}
            value={c.name}
            onChange={(e) => onChange({ ...c, name: e.target.value })}
            descriptionPosition="label"
            description={
              isHeader
                ? "The HTTP header to match against."
                : "The query parameter to match against."
            }
            error={errors?.name?.message}
          />
          {!c.present && (
            <div className="flex flex-col gap-2">
              <div className="flex gap-2">
                <div className="w-28 shrink-0">
                  <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
                    <label htmlFor={`hq-mode-${c.id}`} className="text-gray-11 text-[13px]">
                      Mode
                    </label>
                    <Select
                      value={c.mode ?? "exact"}
                      onValueChange={(v) => onChange({ ...c, mode: v as StringMatchMode })}
                    >
                      <SelectTrigger
                        id={`hq-mode-${c.id}`}
                        rightIcon={
                          <ChevronDown className="absolute right-2" iconSize="md-medium" />
                        }
                      >
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {STRING_MATCH_MODES.map((m) => (
                          <SelectItem key={m.value} value={m.value}>
                            {m.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </fieldset>
                </div>
                <FormInput
                  label="Value"
                  required
                  placeholder="Expected value"
                  value={c.value ?? ""}
                  onChange={(e) => onChange({ ...c, value: e.target.value })}
                  className="flex-1"
                  error={
                    ((c.mode ?? "exact") === "regex"
                      ? validateRegexSyntax(c.value ?? "")
                      : undefined) ?? errors?.value?.message
                  }
                />
              </div>
              {(c.mode ?? "exact") === "regex" && (
                <RegexGenerateInput
                  conditionType={conditionType}
                  onGenerated={(pattern) => onChange({ ...c, value: pattern })}
                />
              )}
            </div>
          )}
        </div>
      );
    })
    .exhaustive();
}

function RegexGenerateInput({
  conditionType,
  onGenerated,
}: {
  conditionType: "path" | "header" | "queryParam";
  onGenerated: (pattern: string) => void;
}) {
  const [prompt, setPrompt] = useState("");
  const generateRegex = trpc.deploy.environmentSettings.sentinel.generateRegex.useMutation({
    onSuccess(data) {
      onGenerated(data.pattern);
      setPrompt("");
    },
    onError(error) {
      toast.error(error.message || "Failed to generate regex pattern", {
        duration: 5000,
        position: "top-right",
      });
    },
  });

  return (
    <div className="flex gap-2 items-end">
      <FormInput
        label="Describe what to match"
        placeholder={
          conditionType === "path"
            ? "e.g. all API routes under /api/v2"
            : conditionType === "header"
              ? "e.g. bearer tokens"
              : "e.g. numeric values only"
        }
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        className="flex-1"
        onKeyDown={(e) => {
          if (e.key === "Enter" && prompt.trim().length >= 3) {
            e.preventDefault();
            generateRegex.mutate({ query: prompt, conditionType });
          }
        }}
      />
      <Button
        type="button"
        variant="primary"
        size="lg"
        className="shrink-0"
        disabled={prompt.trim().length < 3 || generateRegex.isLoading}
        loading={generateRegex.isLoading}
        onClick={() => generateRegex.mutate({ query: prompt, conditionType })}
      >
        <Sparkle3 iconSize="sm-regular" />
        Generate
      </Button>
    </div>
  );
}
