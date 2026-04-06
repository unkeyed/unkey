"use client";

import { trpc } from "@/lib/trpc/client";
import type { StringMatchMode } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { ChevronDown, Plus, Sparkle3, Trash } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Separator,
  toast,
} from "@unkey/ui";
import { FormDescription } from "@unkey/ui/src/components/form/form-helpers";
import { Fragment, useState } from "react";
import type { MatchConditionFormValues } from "./schema";

const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];

const STRING_MATCH_MODES: { value: StringMatchMode; label: string }[] = [
  { value: "exact", label: "Exact" },
  { value: "prefix", label: "Prefix" },
  { value: "regex", label: "Regex" },
];

const MATCH_TYPE_OPTIONS: { value: MatchConditionFormValues["type"]; label: string }[] = [
  { value: "path", label: "Path" },
  { value: "method", label: "Method" },
  { value: "header", label: "Header" },
  { value: "queryParam", label: "Query Param" },
];

function validateRegexSyntax(pattern: string): string | undefined {
  if (!pattern) {
    return undefined;
  }
  try {
    new RegExp(pattern);
    return undefined;
  } catch (e) {
    return e instanceof SyntaxError ? e.message : "Invalid regex pattern";
  }
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

function ConditionFields({
  condition,
  onChange,
}: {
  condition: MatchConditionFormValues;
  onChange: (updated: MatchConditionFormValues) => void;
}) {
  return match(condition)
    .with({ type: "path" }, (c) => (
      <div className="flex flex-col gap-2">
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
                <SelectContent className="z-60">
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
            placeholder={c.mode === "regex" ? "^/api/.*" : "/api/v1"}
            value={c.value}
            onChange={(e) => onChange({ ...c, value: e.target.value })}
            className="flex-1"
            description={
              c.mode === "regex"
                ? "Regex pattern for path matching."
                : "The URL path to match against."
            }
            error={c.mode === "regex" ? validateRegexSyntax(c.value) : undefined}
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
    .with({ type: "method" }, (c) => (
      <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
        <legend className="text-gray-11 text-[13px] mb-1.5">Allowed Methods</legend>
        <div
          className="flex flex-wrap gap-1.5"
          role="group"
          aria-describedby={`method-desc-${c.id}`}
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
                className={`px-2 py-0.5 rounded text-xs font-mono border transition-colors ${active
                  ? "bg-accent-3 border-accent-6 text-accent-11"
                  : "bg-grayA-2 border-grayA-4 text-grayA-9 hover:text-gray-12"
                  }`}
              >
                {m}
              </button>
            );
          })}
        </div>
        <FormDescription
          description="Select which HTTP methods this condition matches."
          descriptionId={`method-desc-${c.id}`}
          errorId={`method-error-${c.id}`}
        />
      </fieldset>
    ))
    .with({ type: "header" }, { type: "queryParam" }, (c) => {
      const isHeader = c.type === "header";
      const conditionType = c.type;
      return (
        <div className="flex flex-col gap-2">
          <FormInput
            label={isHeader ? "Header Name" : "Parameter Name"}
            placeholder={isHeader ? "X-Custom-Header" : "param_name"}
            value={c.name}
            onChange={(e) => onChange({ ...c, name: e.target.value })}
            description={
              isHeader
                ? "The HTTP header to match against."
                : "The query parameter to match against."
            }
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
                      <SelectContent className="z-60">
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
                  placeholder="Expected value"
                  value={c.value ?? ""}
                  onChange={(e) => onChange({ ...c, value: e.target.value })}
                  className="flex-1"
                  error={
                    (c.mode ?? "exact") === "regex"
                      ? validateRegexSyntax(c.value ?? "")
                      : undefined
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

function MatchConditionCard({
  condition,
  onChange,
  onDelete,
}: {
  condition: MatchConditionFormValues;
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
                  mode: "exact" as const,
                  value: "",
                }))
                .with("method", () => ({
                  ...base,
                  type: "method" as const,
                  methods: [] as string[],
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
      <ConditionFields condition={condition} onChange={onChange} />
    </div>
  );
}

export function MatchConditionEditor({
  conditions,
  onChange,
}: {
  conditions: MatchConditionFormValues[];
  onChange: (conditions: MatchConditionFormValues[]) => void;
}) {
  const hasConditions = conditions.length > 0;
  const [expanded, setExpanded] = useState(hasConditions);

  const addFirstCondition = () => {
    onChange([{ id: crypto.randomUUID(), type: "path", mode: "exact", value: "" }]);
    setExpanded(true);
  };

  if (!expanded && !hasConditions) {
    return (
      <div className="border-t border-grayA-4 px-8 py-4">
        <button
          type="button"
          onClick={addFirstCondition}
          className="flex items-center gap-2 text-[13px] text-gray-11 hover:text-gray-12 transition-colors"
        >
          <Plus iconSize="sm-regular" />
          <span>Add match conditions</span>
          <span className="text-gray-9">to restrict which requests this policy applies to</span>
        </button>
      </div>
    );
  }

  return (
    <div className="border-t border-grayA-4">
      <div className="px-8 pt-6 flex items-start justify-between bg-grayA-2">
        <div>
          <span id="match-conditions-label" className="text-gray-11 text-[13px]">
            Match Conditions
          </span>
          <FormDescription
            description={
              <span>
                All conditions must match (<span className="text-gray-12 font-medium">AND</span>{" "}
                logic).
              </span>
            }
            descriptionId="match-conditions-desc"
            errorId="match-conditions-error"
          />
        </div>
        {hasConditions && (
          <button
            type="button"
            onClick={() => {
              onChange([]);
              setExpanded(false);
            }}
            className="text-[12px] text-gray-9 hover:text-gray-12 transition-colors mt-0.5"
          >
            Clear all
          </button>
        )}
      </div>

      <div className="flex flex-col gap-8 px-8 pt-3 max-h-100 overflow-y-auto bg-grayA-2">
        {conditions.map((cond) => (
          <Fragment
            key={cond.id}
          ><MatchConditionCard

              condition={cond}
              onChange={(updated) => onChange(conditions.map((c) => (c.id === updated.id ? updated : c)))}
              onDelete={(id) => {
                const next = conditions.filter((c) => c.id !== id);
                onChange(next);
                if (next.length === 0) {
                  setExpanded(false);
                }
              }} /><Separator /></Fragment>

        ))}
      </div>

      <div className="flex py-6 px-8 bg-grayA-2">
        <Button
          type="button"
          variant="outline"
          size="md"
          className="font-medium"
          onClick={() =>
            onChange([
              ...conditions,
              { id: crypto.randomUUID(), type: "path", mode: "exact", value: "" },
            ])
          }
        >
          <Plus iconSize="sm-regular" />
          Add Condition
        </Button>
      </div>
    </div>
  );
}
