"use client";

import Link from "next/link";
import { IconArrowUpRightOutline12 } from "nucleo-ui-outline-12";

export const PAUSED_DOCS_URL =
  "https://unkey.com/docs/platform/workspaces/billing#while-compute-is-paused";

export function pausedBody(budgetLabel?: string): string {
  const cap = budgetLabel ? `your ${budgetLabel} spend budget` : "your spend budget";
  return `Workloads stopped because you reached ${cap}. Raise or remove the budget to start them again.`;
}

export function ComputePausedBadge() {
  return (
    <span className="rounded-full bg-warning-9 px-2 py-0.5 font-medium text-[11px] text-black">
      Paused
    </span>
  );
}

export function PausedDocsLink() {
  return (
    <Link
      href={PAUSED_DOCS_URL}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-0.5 font-medium text-gray-12 underline decoration-grayA-6 underline-offset-2 hover:decoration-grayA-8"
    >
      Learn more
      <IconArrowUpRightOutline12 />
    </Link>
  );
}
