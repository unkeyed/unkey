"use client";

import { cn } from "@/lib/utils";
import { Plus } from "@unkey/icons";
import { motion } from "framer-motion";
import Link from "next/link";
import { useId } from "react";
import { ApiRow, ComputeRow } from "./usage-rows";
import { useMinimised } from "./use-minimised";
import type { UsageSummary } from "./use-usage-summary";

const TOGGLE = { duration: 0.2, ease: "easeInOut" } as const;

export function UsagePanel({ summary }: { summary: UsageSummary }) {
  const [minimised, setMinimised] = useMinimised();
  const folded = minimised && !summary.atRisk;
  const bodyId = useId();

  return (
    <div className="group relative overflow-hidden rounded-lg border border-grayA-4 bg-white shadow-xs dark:bg-grayA-3">
      {folded ? null : (
        <Link
          href={summary.href}
          aria-label="Usage"
          className="absolute inset-0 rounded-lg hover:bg-grayA-2"
        />
      )}
      <div
        className={cn(
          "pointer-events-none relative flex items-center gap-2 font-medium text-[11px]",
          !folded && "px-2.5 pt-2 pb-1.5",
        )}
      >
        {folded ? null : <span className="min-w-0 truncate text-gray-12">Usage</span>}
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
                ? "w-full gap-2 px-2.5 pt-2 pb-1.5 text-left font-medium text-[11px] transition-colors hover:bg-grayA-2"
                : "-mr-1 h-5 flex-1 justify-end pr-1 pl-2",
            )}
          >
            {folded ? <span className="flex-1 truncate text-gray-12">Usage</span> : null}
            <Toggle rotated={!folded} />
          </button>
        )}
      </div>

      <motion.div
        id={bodyId}
        inert={folded || undefined}
        initial={false}
        animate={{ height: folded ? 0 : "auto", opacity: folded ? 0 : 1 }}
        transition={TOGGLE}
        className="pointer-events-none relative overflow-hidden"
      >
        <div className="flex flex-col gap-2.5 px-2.5 pt-1 pb-2.5">
          {summary.compute === null ? null : <ComputeRow measured={summary.compute} />}
          {summary.api === null ? null : <ApiRow measured={summary.api} />}
        </div>
      </motion.div>
    </div>
  );
}

function Toggle({ rotated }: { rotated: boolean }) {
  return (
    <motion.span
      className="flex shrink-0"
      initial={false}
      animate={{ rotate: rotated ? 45 : 0 }}
      transition={TOGGLE}
    >
      <Plus iconSize="sm-regular" />
    </motion.span>
  );
}
