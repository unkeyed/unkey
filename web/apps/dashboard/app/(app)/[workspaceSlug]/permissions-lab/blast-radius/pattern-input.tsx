"use client";

/**
 * Validated resource-path input shared by the Explore and Compare tabs.
 * Validation runs on every keystroke via validateResourcePath; errors render
 * inline under the field so the coverage panel can stay focused on results.
 */

import { Input } from "@unkey/ui";
import { useId } from "react";
import { urnString } from "../lib/mock-data";
import { validateResourcePath } from "../lib/urn";

/** null when the value is empty or valid; the grammar error otherwise. */
export function patternError(raw: string): string | null {
  const trimmed = raw.trim();
  if (trimmed === "") {
    return null;
  }
  return validateResourcePath(trimmed);
}

export function PatternInput({
  label,
  value,
  onChange,
  placeholder,
  showUrnPreview = false,
}: {
  label: string;
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
  showUrnPreview?: boolean;
}) {
  const id = useId();
  const trimmed = value.trim();
  const error = patternError(value);

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-xs font-medium uppercase tracking-wide text-gray-10">
        {label}
      </label>
      <Input
        id={id}
        value={value}
        variant={error !== null ? "error" : "default"}
        onChange={(event) => onChange(event.currentTarget.value)}
        placeholder={placeholder ?? "keyspaces/*"}
        autoComplete="off"
        spellCheck={false}
        className="font-mono"
      />
      {error !== null ? (
        <p className="text-xs text-error-11">{error}</p>
      ) : showUrnPreview && trimmed !== "" ? (
        <p className="font-mono text-xs text-gray-9 truncate" title={urnString(trimmed)}>
          {urnString(trimmed)}
        </p>
      ) : null}
    </div>
  );
}
