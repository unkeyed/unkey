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
    <div className="group overflow-hidden rounded-lg border border-grayA-4 bg-white shadow-xs dark:bg-grayA-3">
      <div
        className={cn(
          "flex items-center gap-2 font-medium text-[11px]",
          !folded && "px-2.5 pt-2 pb-1.5",
        )}
      >
        {folded ? null : (
          <Link href={summary.href} className="flex min-w-0 items-center gap-1 text-gray-12">
            <span className="truncate">Usage</span>
            <span
              aria-hidden="true"
              className="-translate-x-1 shrink-0 text-gray-11 opacity-0 transition-[opacity,transform] duration-150 ease-out group-hover:translate-x-0 group-hover:opacity-100 motion-reduce:transition-none"
            >
              ↗
            </span>
          </Link>
        )}
        {summary.atRisk ? null : (
          <button
            type="button"
            onClick={() => setMinimised(!folded)}
            aria-expanded={!folded}
            aria-controls={bodyId}
            aria-label={folded ? "Show usage detail" : "Hide usage detail"}
            className={cn(
              "flex items-center rounded text-gray-9 hover:text-gray-12",
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
        className="overflow-hidden"
      >
        <Link href={summary.href} className="block">
          <div className="flex flex-col gap-2.5 px-2.5 pt-1 pb-2.5">
            {summary.compute === null ? null : <ComputeRow measured={summary.compute} />}
            {summary.api === null ? null : <ApiRow measured={summary.api} />}
          </div>
        </Link>
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
