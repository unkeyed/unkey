"use client";

import { IconPlusOutline12 } from "@unkey/icons";
import { cn } from "cn";
import Link from "next/link";
import { useId } from "react";
import { ApiRow, ComputeRow } from "./usage-rows";
import { useMinimised } from "./use-minimised";
import type { UsageSummary } from "./use-usage-summary";

export function UsagePanel({ summary }: { summary: UsageSummary }) {
  const [minimised, setMinimised] = useMinimised();
  const folded = minimised && !summary.atRisk;
  const bodyId = useId();

  return (
    <div className="relative overflow-hidden rounded-lg border">
      {folded ? null : (
        <Link
          href={summary.href}
          aria-label="Usage"
          className="absolute inset-0 rounded-lg outline-hidden focus-visible:ring-2 focus-visible:ring-grayA-7"
        />
      )}
      <div
        className={cn(
          "pointer-events-none relative flex items-center gap-2 text-2xs",
          !folded && "px-2.5 pt-2 pb-1.5",
        )}
      >
        {folded ? null : <span className="min-w-0 truncate font-medium text-gray-12">Usage</span>}
        {summary.atRisk ? null : (
          <button
            type="button"
            onClick={() => setMinimised(!folded)}
            aria-expanded={!folded}
            aria-controls={bodyId}
            aria-label={folded ? "Show usage detail" : "Hide usage detail"}
            className={cn(
              "pointer-events-auto flex items-center rounded text-gray-9 hover:text-gray-12",
              folded
                ? "w-full gap-2 px-2.5 pt-2 pb-1.5 text-left text-2xs transition-colors hover:bg-grayA-2"
                : "-mr-1 ml-auto size-5 justify-center",
            )}
          >
            {folded ? (
              <span className="flex-1 truncate font-medium text-gray-12">Usage</span>
            ) : null}
            <Toggle rotated={!folded} />
          </button>
        )}
      </div>

      <div
        id={bodyId}
        inert={folded || undefined}
        className={cn(
          "pointer-events-none relative grid transition-[grid-template-rows,opacity] duration-200 ease-[ease-in-out]",
          folded ? "grid-rows-[0fr] opacity-0" : "grid-rows-[1fr] opacity-100",
        )}
      >
        <div className="min-h-0 overflow-hidden">
          <div className="flex flex-col gap-2.5 px-2.5 pt-1 pb-2.5">
            {summary.compute === null ? null : <ComputeRow measured={summary.compute} />}
            {summary.api === null ? null : <ApiRow measured={summary.api} />}
          </div>
        </div>
      </div>
    </div>
  );
}

function Toggle({ rotated }: { rotated: boolean }) {
  return (
    <span
      className={cn(
        "flex shrink-0 transition-transform duration-200 ease-[ease-in-out]",
        rotated && "rotate-45",
      )}
    >
      <IconPlusOutline12 />
    </span>
  );
}
