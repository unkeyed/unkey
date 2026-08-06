"use client";

/**
 * Explore mode: one pattern input plus one-click examples on the left, the
 * live coverage panel on the right. While the input is invalid the last valid
 * result stays visible, dimmed behind an inline error notice, so the user
 * never loses their bearings mid-edit.
 */

import { Button, Empty, cn } from "@unkey/ui";
import { useEffect, useState } from "react";
import { CoveragePanel } from "./coverage-panel";
import { PatternInput, patternError } from "./pattern-input";

const EXAMPLE_PATTERNS = [
  "keyspaces/*",
  "keyspaces/ks_payments_prod/**",
  "keyspaces/*/keys/*",
  "projects/proj_storefront/**",
  "**",
] as const;

const DEFAULT_PATTERN = "keyspaces/*";

export function ExploreTab() {
  const [pattern, setPattern] = useState<string>(DEFAULT_PATTERN);
  const trimmed = pattern.trim();
  const error = patternError(pattern);

  const [lastValid, setLastValid] = useState<string>(DEFAULT_PATTERN);
  useEffect(() => {
    if (trimmed !== "" && error === null) {
      setLastValid(trimmed);
    }
  }, [trimmed, error]);

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[minmax(280px,380px)_1fr] gap-8 lg:gap-12 items-start">
      <div className="flex flex-col gap-6">
        <PatternInput
          label="Resource pattern"
          value={pattern}
          onChange={setPattern}
          placeholder="keyspaces/*"
          showUrnPreview
        />
        <div className="flex flex-col gap-2">
          <span className="text-xs font-medium uppercase tracking-wide text-gray-10">Try one</span>
          <div className="flex flex-wrap gap-2">
            {EXAMPLE_PATTERNS.map((example) => (
              <Button
                key={example}
                variant="outline"
                onClick={() => setPattern(example)}
                className={cn(
                  "font-mono text-xs",
                  trimmed === example && "bg-grayA-3 border-grayA-8",
                )}
              >
                {example}
              </Button>
            ))}
          </div>
        </div>
        <p className="text-xs text-gray-10">
          Coverage updates on every keystroke against every concrete resource in the ACME mock
          workspace.
        </p>
      </div>

      <div className="min-w-0">
        {trimmed === "" ? (
          <Empty>
            <Empty.Title>No pattern yet</Empty.Title>
            <Empty.Description>
              Type a resource path on the left, or click an example, to see everything it would
              cover.
            </Empty.Description>
          </Empty>
        ) : error !== null ? (
          <div className="flex flex-col gap-5">
            <div className="rounded-lg border border-errorA-3 bg-errorA-2 px-4 py-3 flex flex-col gap-1">
              <span className="text-[13px] font-medium text-error-11">Invalid pattern</span>
              <span className="text-xs text-error-11">{error}</span>
              <span className="text-xs text-gray-10">
                Results below still show the last valid pattern{" "}
                <span className="font-mono text-gray-11">{lastValid}</span>.
              </span>
            </div>
            <CoveragePanel key={lastValid} pattern={lastValid} className="opacity-50" />
          </div>
        ) : (
          <CoveragePanel key={trimmed} pattern={trimmed} />
        )}
      </div>
    </div>
  );
}
