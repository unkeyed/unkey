"use client";

import { Plus, Trash } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import { SimpleSelect } from "./simple-select";
import type { MatchFormData, MatchType, StringMatchMode } from "./types";

const MATCH_TYPE_OPTIONS = [
  { value: "path", label: "Path" },
  { value: "method", label: "Method" },
  { value: "header", label: "Header" },
  { value: "queryParam", label: "Query Param" },
];

const STRING_MATCH_OPTIONS = [
  { value: "exact", label: "Exact" },
  { value: "prefix", label: "Prefix" },
  { value: "regex", label: "Regex" },
];

const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];

export function MatchEditor({
  matches,
  onChange,
}: {
  matches: MatchFormData[];
  onChange: (matches: MatchFormData[]) => void;
}) {
  const add = () => {
    onChange([
      ...matches,
      { id: crypto.randomUUID(), type: "path", pathMode: "prefix", pathValue: "/" },
    ]);
  };

  const update = (id: string, updates: Partial<MatchFormData>) => {
    onChange(matches.map((m) => (m.id === id ? { ...m, ...updates } : m)));
  };

  const remove = (id: string) => {
    onChange(matches.filter((m) => m.id !== id));
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <p className="text-xs font-medium text-gray-11 uppercase tracking-wider">
          Match Conditions {matches.length > 0 && <span className="text-grayA-8">(AND)</span>}
        </p>
        <Button variant="ghost" size="sm" onClick={add}>
          <Plus className="size-3" />
          Add
        </Button>
      </div>
      {matches.length === 0 && (
        <p className="text-xs text-grayA-8 py-1">No conditions — policy applies to all requests.</p>
      )}
      {matches.map((m) => (
        <div key={m.id} className="flex gap-2 rounded-lg border border-grayA-3 bg-gray-1 p-3">
          <div className="flex-1 flex flex-col gap-2">
            <SimpleSelect
              label="Type"
              value={m.type}
              options={MATCH_TYPE_OPTIONS}
              onChange={(v) => update(m.id, { type: v as MatchType })}
            />
            <MatchFields match={m} onUpdate={(updates) => update(m.id, updates)} />
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="text-grayA-8 hover:text-red-10 shrink-0 self-start mt-5"
            onClick={() => remove(m.id)}
          >
            <Trash className="size-3" />
          </Button>
        </div>
      ))}
    </div>
  );
}

function MatchFields({
  match,
  onUpdate,
}: {
  match: MatchFormData;
  onUpdate: (updates: Partial<MatchFormData>) => void;
}) {
  switch (match.type) {
    case "path":
      return (
        <div className="flex items-end gap-2">
          <SimpleSelect
            label="Mode"
            value={match.pathMode ?? "prefix"}
            options={STRING_MATCH_OPTIONS}
            onChange={(v) => onUpdate({ pathMode: v as StringMatchMode })}
          />
          <div className="flex-1">
            <FormInput
              label="Value"
              value={match.pathValue ?? ""}
              placeholder="/api/v1"
              onChange={(e) => onUpdate({ pathValue: e.target.value })}
            />
          </div>
        </div>
      );
    case "method":
      return (
        <div className="flex flex-col gap-1.5">
          <span className="text-xs font-medium text-gray-11">Methods</span>
          <div className="flex flex-wrap gap-1.5">
            {HTTP_METHODS.map((method) => {
              const selected = match.methods?.includes(method) ?? false;
              return (
                <button
                  key={method}
                  type="button"
                  className={`px-2 py-1 rounded text-xs font-mono transition-colors border ${
                    selected
                      ? "bg-accent-3 border-accent-7 text-accent-11"
                      : "bg-gray-1 border-grayA-4 text-grayA-9 hover:border-grayA-6"
                  }`}
                  onClick={() => {
                    const methods = match.methods ?? [];
                    onUpdate({
                      methods: selected
                        ? methods.filter((m) => m !== method)
                        : [...methods, method],
                    });
                  }}
                >
                  {method}
                </button>
              );
            })}
          </div>
        </div>
      );
    case "header":
      return (
        <div className="flex flex-col gap-2">
          <FormInput
            label="Header Name"
            value={match.headerName ?? ""}
            placeholder="X-API-Version"
            onChange={(e) => onUpdate({ headerName: e.target.value })}
          />
          <label className="flex items-center gap-1.5 text-xs text-gray-11 cursor-pointer">
            <input
              type="checkbox"
              checked={match.headerPresent ?? false}
              onChange={(e) => onUpdate({ headerPresent: e.target.checked })}
              className="rounded"
            />
            Match presence only
          </label>
          {!match.headerPresent && (
            <div className="flex items-end gap-2">
              <SimpleSelect
                label="Mode"
                value={match.headerMode ?? "exact"}
                options={STRING_MATCH_OPTIONS}
                onChange={(v) => onUpdate({ headerMode: v as StringMatchMode })}
              />
              <div className="flex-1">
                <FormInput
                  label="Value"
                  value={match.headerValue ?? ""}
                  onChange={(e) => onUpdate({ headerValue: e.target.value })}
                />
              </div>
            </div>
          )}
        </div>
      );
    case "queryParam":
      return (
        <div className="flex flex-col gap-2">
          <FormInput
            label="Param Name"
            value={match.queryParamName ?? ""}
            placeholder="version"
            onChange={(e) => onUpdate({ queryParamName: e.target.value })}
          />
          <label className="flex items-center gap-1.5 text-xs text-gray-11 cursor-pointer">
            <input
              type="checkbox"
              checked={match.queryParamPresent ?? false}
              onChange={(e) => onUpdate({ queryParamPresent: e.target.checked })}
              className="rounded"
            />
            Match presence only
          </label>
          {!match.queryParamPresent && (
            <div className="flex items-end gap-2">
              <SimpleSelect
                label="Mode"
                value={match.queryParamMode ?? "exact"}
                options={STRING_MATCH_OPTIONS}
                onChange={(v) => onUpdate({ queryParamMode: v as StringMatchMode })}
              />
              <div className="flex-1">
                <FormInput
                  label="Value"
                  value={match.queryParamValue ?? ""}
                  onChange={(e) => onUpdate({ queryParamValue: e.target.value })}
                />
              </div>
            </div>
          )}
        </div>
      );
  }
}
