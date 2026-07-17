"use client";

/**
 * Compare mode: two patterns, "current" and "proposed", diffed by set
 * difference over their concrete coverage. Gained resources render
 * success-toned with a plus, lost ones error-toned with a minus, and the
 * unchanged overlap collapses to a count.
 */

import { cn } from "@unkey/ui";
import { useMemo, useState } from "react";
import { type ConcreteResource, coverage } from "../lib/mock-data";
import { ExpanderButton, ResourceRow } from "./coverage-panel";
import { PatternInput, patternError } from "./pattern-input";

const MAX_VISIBLE_ROWS = 8;

function DiffSection({
  title,
  resources,
  tone,
}: {
  title: string;
  resources: ConcreteResource[];
  tone: "gained" | "lost";
}) {
  const [expanded, setExpanded] = useState(false);
  if (resources.length === 0) {
    return null;
  }
  const visible = expanded ? resources : resources.slice(0, MAX_VISIBLE_ROWS);
  const hidden = resources.length - visible.length;
  const overflows = resources.length > MAX_VISIBLE_ROWS;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-baseline gap-2">
        <span
          className={cn(
            "text-[13px] font-medium",
            tone === "gained" ? "text-success-11" : "text-error-11",
          )}
        >
          {title}
        </span>
        <span className="text-xs text-gray-10 tabular-nums">
          {resources.length} {resources.length === 1 ? "resource" : "resources"}
        </span>
      </div>
      <div className="flex flex-col gap-1">
        {visible.map((resource) => (
          <ResourceRow key={resource.path} resource={resource} tone={tone} />
        ))}
      </div>
      {overflows && (
        <ExpanderButton
          hidden={hidden}
          expanded={expanded}
          onToggle={() => setExpanded((prev) => !prev)}
        />
      )}
    </div>
  );
}

function Stat({
  label,
  value,
  className,
}: {
  label: string;
  value: number;
  className?: string;
}) {
  return (
    <div className="flex flex-col">
      <span className={cn("text-2xl font-semibold tabular-nums text-gray-12", className)}>
        {value}
      </span>
      <span className="text-xs text-gray-10">{label}</span>
    </div>
  );
}

export function CompareTab() {
  const [current, setCurrent] = useState("keyspaces/ks_payments_prod/**");
  const [proposed, setProposed] = useState("keyspaces/*/keys/*");

  const currentTrimmed = current.trim();
  const proposedTrimmed = proposed.trim();
  const currentError = patternError(current);
  const proposedError = patternError(proposed);
  const ready =
    currentTrimmed !== "" &&
    proposedTrimmed !== "" &&
    currentError === null &&
    proposedError === null;

  const diff = useMemo(() => {
    if (!ready) {
      return null;
    }
    const currentResources = coverage(currentTrimmed);
    const proposedResources = coverage(proposedTrimmed);
    const currentPaths = new Set(currentResources.map((r) => r.path));
    const proposedPaths = new Set(proposedResources.map((r) => r.path));
    return {
      unchanged: currentResources.filter((r) => proposedPaths.has(r.path)),
      gained: proposedResources.filter((r) => !currentPaths.has(r.path)),
      lost: currentResources.filter((r) => !proposedPaths.has(r.path)),
    };
  }, [ready, currentTrimmed, proposedTrimmed]);

  return (
    <div className="flex flex-col gap-8">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <PatternInput
          label="Current pattern"
          value={current}
          onChange={setCurrent}
          placeholder="keyspaces/ks_payments_prod/**"
        />
        <PatternInput
          label="Proposed pattern"
          value={proposed}
          onChange={setProposed}
          placeholder="keyspaces/*/keys/*"
        />
      </div>

      {diff === null ? (
        <div className="rounded-lg border border-grayA-4 bg-grayA-2 px-5 py-4">
          <p className="text-sm text-gray-11">
            Enter a valid pattern on both sides to see what the change would gain and lose.
          </p>
        </div>
      ) : (
        <div className="flex flex-col gap-6">
          <div className="rounded-lg border border-grayA-4 bg-grayA-2 px-5 py-4 flex flex-col gap-4">
            <p className="text-sm text-gray-12">
              {diff.gained.length === 0 && diff.lost.length === 0 ? (
                <>
                  No change in coverage. Both patterns cover the same{" "}
                  <span className="font-semibold tabular-nums">{diff.unchanged.length}</span>{" "}
                  {diff.unchanged.length === 1 ? "resource" : "resources"}.
                </>
              ) : (
                <>
                  Proposed pattern gains{" "}
                  <span className="font-semibold text-success-11 tabular-nums">
                    {diff.gained.length}
                  </span>{" "}
                  and loses{" "}
                  <span className="font-semibold text-error-11 tabular-nums">
                    {diff.lost.length}
                  </span>{" "}
                  resources.
                </>
              )}
            </p>
            <div className="flex gap-8">
              <Stat label="Unchanged" value={diff.unchanged.length} />
              <Stat label="Gained" value={diff.gained.length} className="text-success-11" />
              <Stat label="Lost" value={diff.lost.length} className="text-error-11" />
            </div>
          </div>

          <DiffSection title="Gained" resources={diff.gained} tone="gained" />
          <DiffSection title="Lost" resources={diff.lost} tone="lost" />

          {diff.unchanged.length > 0 && (diff.gained.length > 0 || diff.lost.length > 0) && (
            <p className="text-xs text-gray-10 tabular-nums">
              {diff.unchanged.length}{" "}
              {diff.unchanged.length === 1 ? "resource is" : "resources are"} covered by both
              patterns and unaffected by this change.
            </p>
          )}
        </div>
      )}
    </div>
  );
}
