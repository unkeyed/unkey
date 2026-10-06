"use client";

import {
  IconArrowUpRightOutline12,
  IconCheckOutline12,
  IconMinusOutline12,
  IconXmarkOutline12,
} from "@unkey/icons";
import { CopyButton, Loading, Skeleton } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";
import { type ReactNode, useLayoutEffect, useRef, useState } from "react";
import { type LogTone, type StageState, emptyLogCopy, formatOffset } from "./run-model";
import { BUILD_LOG_LIMIT, type DeployRun } from "./use-deploy-run";

export const stageGlyph: Record<StageState, ReactNode> = {
  done: (
    <span className="grid size-4 place-items-center rounded-full bg-success-9 text-white">
      <IconCheckOutline12 className="size-2.5" />
    </span>
  ),
  active: <Loading size={14} />,
  failed: (
    <span className="grid size-4 place-items-center rounded-full bg-error-9 text-white">
      <IconXmarkOutline12 className="size-2.5" />
    </span>
  ),
  waiting: <span className="size-4 rounded-full border-[1.5px] border-dashed border-gray-8" />,
  skipped: (
    <span className="grid size-4 place-items-center rounded-full bg-grayA-3 text-gray-9">
      <IconMinusOutline12 className="size-2.5" />
    </span>
  ),
};

const toneClass: Record<LogTone, string> = {
  plain: "text-gray-12",
  error: "bg-error-3 text-error-11",
  warn: "bg-warning-3 text-warning-11",
};

const FOLLOW_SLACK_PX = 24;

function useAutoTail(lineCount: number) {
  const ref = useRef<HTMLDivElement>(null);
  const [following, setFollowing] = useState(true);

  // biome-ignore lint/correctness/useExhaustiveDependencies: lineCount is the trigger to re-tail.
  useLayoutEffect(() => {
    const el = ref.current;
    if (el && following) {
      el.scrollTop = el.scrollHeight;
    }
  }, [lineCount, following]);

  const onScroll = () => {
    const el = ref.current;
    if (el) {
      setFollowing(el.scrollHeight - el.scrollTop - el.clientHeight < FOLLOW_SLACK_PX);
    }
  };

  return { ref, following, onScroll, resume: () => setFollowing(true) };
}

export function LogBox({ run, className }: { run: DeployRun; className?: string }) {
  const lines = run.groups.flatMap((group) => group.lines);
  const tail = useAutoTail(lines.length + run.groups.length);
  const errors = lines.filter((line) => line.tone === "error").length;
  const warnings = lines.filter((line) => line.tone === "warn").length;
  const text = run.groups
    .map((group) =>
      [group.title, ...group.lines.map((l) => `${formatOffset(l.offsetMs)}  ${l.text}`)].join("\n"),
    )
    .join("\n");

  return (
    <div
      className={cn(
        "flex min-h-0 flex-col overflow-hidden rounded-lg border border-grayA-4 bg-gray-2",
        className,
      )}
    >
      <div className="flex h-9 shrink-0 items-center gap-3 border-b border-grayA-4 px-3 text-xs text-gray-10">
        <span className="tabular-nums">
          {lines.length} {lines.length === 1 ? "line" : "lines"}
        </span>
        {errors > 0 ? <span className="tabular-nums text-error-11">{errors} errors</span> : null}
        {warnings > 0 ? (
          <span className="tabular-nums text-warning-11">{warnings} warnings</span>
        ) : null}
        <span className="ml-auto flex items-center gap-2">
          {tail.following ? (
            <span className="flex items-center gap-1.5">
              <span className="size-1.5 rounded-full bg-success-9" />
              Auto-scroll
            </span>
          ) : (
            <button
              type="button"
              onClick={tail.resume}
              className="rounded-md px-1.5 py-0.5 font-medium text-gray-12 hover:bg-grayA-3"
            >
              Jump to latest
            </button>
          )}
          {lines.length > 0 ? (
            <CopyButton value={text} variant="ghost" size="icon" aria-label="Copy log" />
          ) : null}
        </span>
      </div>
      <div
        ref={tail.ref}
        onScroll={tail.onScroll}
        className="min-h-0 flex-1 overflow-y-auto overscroll-contain py-2 font-mono text-xs leading-5 [scrollbar-width:thin]"
      >
        {run.logsLoading ? (
          <div className="flex flex-col gap-2 px-3">
            <Skeleton className="h-3 w-2/3 rounded" />
            <Skeleton className="h-3 w-1/2 rounded" />
            <Skeleton className="h-3 w-3/5 rounded" />
          </div>
        ) : run.groups.length === 0 ? (
          <EmptyLog run={run} />
        ) : (
          run.groups.map((group) => (
            <div key={group.id}>
              <div
                className={cn(
                  "px-3 pt-2 font-sans text-xs font-medium",
                  group.failed ? "text-error-11" : "text-gray-11",
                )}
              >
                {group.title}
              </div>
              {group.lines.map((line) => (
                <div key={line.id} className={cn("flex gap-3 px-3", toneClass[line.tone])}>
                  <span className="shrink-0 select-none tabular-nums text-gray-9">
                    {formatOffset(line.offsetMs)}
                  </span>
                  <span className="min-w-0 whitespace-pre-wrap break-all">{line.text}</span>
                </div>
              ))}
            </div>
          ))
        )}
        {run.buildLogsCapped ? (
          <p className="px-3 pt-2 font-sans text-xs text-gray-10">
            Showing the first {BUILD_LOG_LIMIT} lines.{" "}
            <Link href={run.deploymentHref} className="underline underline-offset-2">
              View the full log
            </Link>
          </p>
        ) : null}
      </div>
    </div>
  );
}

function EmptyLog({ run }: { run: DeployRun }) {
  const copy = emptyLogCopy(run.source, run.view.outcome);
  return (
    <div className="flex flex-col gap-1 px-3 py-2 font-sans text-xs">
      <p className="text-gray-12">{copy.title}</p>
      <p className="text-gray-10">{copy.reason}</p>
    </div>
  );
}

export function LiveUrl({ run }: { run: DeployRun }) {
  const domain = run.primaryDomain;
  if (!domain) {
    return <span className="text-sm text-gray-10">No domain is assigned yet.</span>;
  }
  return (
    <span className="flex min-w-0 items-center gap-1">
      <a
        href={domain.url}
        title={domain.url}
        target="_blank"
        rel="noopener noreferrer"
        className="flex min-w-0 items-center gap-1 font-mono text-sm text-gray-12 underline decoration-grayA-6 underline-offset-4 hover:decoration-gray-12"
      >
        <span className="truncate">{domain.hostname}</span>
        <IconArrowUpRightOutline12 className="size-3 shrink-0 text-gray-10" />
      </a>
      <CopyButton value={domain.url} variant="ghost" size="icon" aria-label="Copy URL" />
    </span>
  );
}
