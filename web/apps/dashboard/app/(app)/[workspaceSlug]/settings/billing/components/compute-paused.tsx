"use client";

import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { ArrowUpRight, TriangleWarning } from "@unkey/icons";
import Link from "next/link";
import { type ReactNode, createContext, useContext, useState } from "react";

export const PAUSED_DOCS_URL =
  "https://unkey.com/docs/platform/workspaces/compute-billing#while-compute-is-paused";

/**
 * Design avenues for the paused state, swapped live by the dev debug bar:
 * - "banner": a bordered notice above the meter (the loud version).
 * - "inline": a single quiet amber row above the meter; the header badge
 *   carries the alarm.
 * - "meter": no separate notice — a pill on the budget label and a one-line
 *   caption, leaning on the red meter to tell the story.
 */
export const PAUSED_VARIANTS = ["banner", "inline", "meter"] as const;
export type PausedVariant = (typeof PAUSED_VARIANTS)[number];

/**
 * One concise sentence, budget interpolated when known. The "stop workloads"
 * mechanics and the full list of resume conditions live in docs (linked
 * alongside) rather than in the sentence.
 */
export function pausedBody(budgetLabel?: string): string {
  const cap = budgetLabel ? `your ${budgetLabel} spend budget` : "your spend budget";
  return `Workloads stopped because you reached ${cap}. Raise or remove the budget to start them again — it takes about a minute.`;
}

/** Solid amber status pill for the card header, next to the plan tag. */
export function ComputePausedBadge() {
  return (
    <span className="rounded-full bg-warning-9 px-2 py-0.5 font-medium text-[11px] text-black">
      Paused
    </span>
  );
}

/** "Learn more" docs link with the standard out-arrow. */
function PausedDocsLink() {
  return (
    <Link
      href={PAUSED_DOCS_URL}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-0.5 font-medium text-gray-12 underline decoration-grayA-6 underline-offset-2 hover:decoration-grayA-8"
    >
      Learn more
      <ArrowUpRight iconSize="sm-regular" />
    </Link>
  );
}

/** Small amber pill for the "meter" avenue, shown inline with the label. */
export function ComputePausedMeterPill() {
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-warningA-3 px-2 py-0.5 font-medium text-[11px] text-warning-11">
      <TriangleWarning iconSize="sm-regular" />
      Paused
    </span>
  );
}

/** One-line amber caption for the "meter" avenue, under the meter. */
export function ComputePausedMeterCaption({ budgetLabel }: { budgetLabel?: string }) {
  return (
    <span className="text-[12px] text-warning-11">
      Workloads stopped at {budgetLabel ?? "your budget"}. Raise or remove it to resume.{" "}
      <PausedDocsLink />
    </span>
  );
}

type NoticeProps = {
  variant: PausedVariant;
  budgetLabel?: string;
  /** SpendBudget's own Edit button, placed by the notice so the dialog stays
   *  owned by SpendBudget. */
  action?: ReactNode;
};

/**
 * The above-the-meter notice for the "banner" and "inline" avenues. Returns
 * null for "meter", where the state is shown on the meter itself.
 */
export function ComputePausedNotice({ variant, budgetLabel, action }: NoticeProps) {
  if (variant === "meter") {
    return null;
  }

  if (variant === "inline") {
    return (
      <div className="flex items-start gap-2.5">
        <TriangleWarning iconSize="md-regular" className="mt-0.5 shrink-0 text-warning-11" />
        <p className="text-[13px] text-gray-11 leading-5">
          <span className="font-medium text-gray-12">Compute paused.</span>{" "}
          {pausedBody(budgetLabel)} <PausedDocsLink />
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-warning-6 bg-warningA-2 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-start gap-2.5">
        <TriangleWarning iconSize="md-regular" className="mt-0.5 shrink-0 text-warning-11" />
        <div className="flex flex-col gap-0.5">
          <span className="font-medium text-[13px] text-gray-12">Compute paused</span>
          <p className="text-[13px] text-gray-11 leading-5">
            {pausedBody(budgetLabel)} <PausedDocsLink />
          </p>
        </div>
      </div>
      {action ? <div className="shrink-0 sm:pl-2">{action}</div> : null}
    </div>
  );
}

/* -------------------------------------------------------------------------- */
/* Dev-only preview plumbing                                                  */
/* -------------------------------------------------------------------------- */

type PausedPreview = {
  /** Force the suspended state on so the paused UI can be previewed without
   *  hitting a real spend cap. */
  forceSuspended: boolean;
  variant: PausedVariant;
  setForceSuspended: (value: boolean) => void;
  setVariant: (value: PausedVariant) => void;
};

const PausedPreviewContext = createContext<PausedPreview | null>(null);

/** Null outside the provider (i.e. in production, where it isn't mounted). */
export function usePausedPreview(): PausedPreview | null {
  return useContext(PausedPreviewContext);
}

export function PausedPreviewProvider({ children }: { children: ReactNode }) {
  const [forceSuspended, setForceSuspended] = useState(false);
  const [variant, setVariant] = useState<PausedVariant>("banner");
  return (
    <PausedPreviewContext.Provider
      value={{ forceSuspended, variant, setForceSuspended, setVariant }}
    >
      {children}
    </PausedPreviewContext.Provider>
  );
}

const VARIANT_LABELS: Record<PausedVariant, string> = {
  banner: "Banner",
  inline: "Inline row",
  meter: "On meter",
};

/**
 * Sticky dev bar for flipping the paused state on and swapping avenues in
 * place. Rendered only in development; delete this and the provider once an
 * avenue is chosen.
 */
export function ComputePausedDebugBar() {
  const preview = usePausedPreview();
  if (!preview) {
    return null;
  }
  return (
    <div className="sticky top-0 z-10 -mx-1 flex flex-wrap items-center gap-x-4 gap-y-2 rounded-lg border border-grayA-4 border-dashed bg-grayA-2 px-4 py-2.5 backdrop-blur">
      <span className="font-medium text-[11px] text-gray-9 uppercase tracking-wide">
        Paused preview
      </span>
      <span className="flex items-center gap-2 text-[13px] text-gray-11">
        <Switch checked={preview.forceSuspended} onCheckedChange={preview.setForceSuspended} />
        Force paused
      </span>
      <div className="flex items-center gap-1">
        {PAUSED_VARIANTS.map((v) => (
          <button
            key={v}
            type="button"
            disabled={!preview.forceSuspended}
            onClick={() => preview.setVariant(v)}
            className={cn(
              "rounded-md px-2.5 py-1 font-medium text-[12px] transition-colors disabled:opacity-40",
              preview.variant === v
                ? "bg-grayA-4 text-gray-12"
                : "text-gray-10 hover:bg-grayA-3 hover:text-gray-12",
            )}
          >
            {VARIANT_LABELS[v]}
          </button>
        ))}
      </div>
    </div>
  );
}
