"use client";

import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
  IconBoltOutline18,
  IconCheckOutline12,
  IconChevronDownOutline12,
  IconChevronExpandYOutline12,
  IconChevronRightOutline12,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import { Button, Loading } from "@unkey/ui";
import { cn } from "cn";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useDeployment } from "../../layout-provider";
import { BUILD_LOG_ENTRIES_SHOWN_MAX, useBuildLogs } from "../../use-build-logs";
import { type BuildLogLine, toBuildLogLines } from "./build-log-lines";

const LINE_HEIGHT_ESTIMATE_PX = 22;
const TAIL_FOLLOW_SLACK_PX = 2 * LINE_HEIGHT_ESTIMATE_PX;
// About three pages of getBuildLogs
const LINES_AHEAD_MIN = 300;
const BUILD_NOT_DONE_STATUSES: ReadonlySet<DeploymentStatus> = new Set([
  "pending",
  "starting",
  "building",
]);

type Props = {
  fixedHeight?: number;
  focusErrorTick?: number;
};

export function DeploymentBuildLogs({ fixedHeight = 500, focusErrorTick = 0 }: Props) {
  const { deployment } = useDeployment();
  const isFailed = deployment.status === "failed";
  const [followsTail, setFollowsTail] = useState(() =>
    BUILD_NOT_DONE_STATUSES.has(deployment.status),
  );
  const [pendingJump, setPendingJump] = useState<"error" | "latest" | null>(null);
  const readsToEnd = isFailed || followsTail || pendingJump !== null;

  const logs = useBuildLogs(deployment, { readsToEnd });
  const entries = logs.data?.entries;
  const hasMore = logs.data?.hasMore === true;

  const [expandedRunKeys, setExpandedRunKeys] = useState<ReadonlySet<string>>(new Set());
  const lines = useMemo(
    () => toBuildLogLines(entries ?? [], expandedRunKeys),
    [entries, expandedRunKeys],
  );
  const lastErrorIndex = lines.findLastIndex(
    (line) => line.kind === "entry" && line.tone === "error",
  );

  const scrollRef = useRef<HTMLDivElement>(null);
  const lastScrollTop = useRef(0);
  const virtualizer = useVirtualizer({
    count: hasMore ? lines.length + 1 : lines.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => LINE_HEIGHT_ESTIMATE_PX,
    overscan: 20,
  });

  const totalSize = virtualizer.getTotalSize();
  const lastVisibleIndex = virtualizer.getVirtualItems().at(-1)?.index ?? 0;
  const isAtBottom =
    (virtualizer.scrollOffset ?? 0) + (virtualizer.scrollRect?.height ?? 0) >=
    totalSize - TAIL_FOLLOW_SLACK_PX;
  const wantsNextPage = readsToEnd || lines.length - lastVisibleIndex < LINES_AHEAD_MIN;

  const { isFetching, isError, refetch } = logs;
  useEffect(() => {
    if (wantsNextPage && hasMore && !isFetching && !isError) {
      void refetch();
    }
  }, [wantsNextPage, hasMore, isFetching, isError, refetch]);

  useEffect(() => {
    const el = scrollRef.current;
    if (followsTail && el) {
      // Past the end on purpose, the browser clamps it to the bottom
      el.scrollTop = totalSize;
    }
  }, [followsTail, totalSize]);

  useEffect(() => {
    if (pendingJump === null || hasMore) {
      return;
    }
    setPendingJump(null);
    if (pendingJump === "latest") {
      setFollowsTail(true);
    } else if (lastErrorIndex >= 0) {
      setFollowsTail(false);
      virtualizer.scrollToIndex(lastErrorIndex, { align: "center" });
    }
  }, [pendingJump, hasMore, lastErrorIndex, virtualizer]);

  useEffect(() => {
    if (focusErrorTick > 0) {
      setPendingJump("error");
    }
  }, [focusErrorTick]);

  const expandRun = (runKey: string) => {
    setFollowsTail(false);
    setExpandedRunKeys((previous) => new Set(previous).add(runKey));
  };

  if (logs.isLoading) {
    return (
      <div className="flex flex-col">
        <div className="flex h-11 items-center gap-2 border-b border-grayA-3 px-4 text-xs text-gray-10">
          <Loading size={12} />
          Loading build logs
        </div>
        <div style={{ height: fixedHeight }} />
      </div>
    );
  }
  if (logs.isError && !logs.data) {
    return (
      <BuildLogsMessage>
        <IconTriangleWarningOutline18 className="size-3.5 text-error-11" />
        Failed to load build logs
      </BuildLogsMessage>
    );
  }
  if (lines.length === 0) {
    return <BuildLogsMessage>No build logs yet</BuildLogsMessage>;
  }

  const entriesTotal = logs.data.entriesTotal;
  const isTruncated = entriesTotal > BUILD_LOG_ENTRIES_SHOWN_MAX;

  return (
    <div className="flex flex-col">
      <div className="flex h-11 items-center justify-between gap-4 border-b border-grayA-3 px-4">
        <span className="text-xs text-gray-11 tabular-nums">
          {isTruncated
            ? `Showing the last ${BUILD_LOG_ENTRIES_SHOWN_MAX.toLocaleString()} of ${entriesTotal.toLocaleString()} entries`
            : `${entriesTotal.toLocaleString()}${hasMore ? "+" : ""} ${entriesTotal === 1 && !hasMore ? "entry" : "entries"}`}
        </span>
        <div className="flex items-center gap-2">
          {(lastErrorIndex >= 0 || (isFailed && hasMore)) && (
            <Button
              variant="outline"
              size="sm"
              className="[&_svg]:size-3.5"
              loading={pendingJump === "error"}
              onClick={() => setPendingJump("error")}
            >
              <IconTriangleWarningOutline18 className="text-error-11" />
              Jump to error
            </Button>
          )}
          {!followsTail && (hasMore || !isAtBottom) && (
            <Button
              variant="outline"
              size="sm"
              className="[&_svg]:size-3"
              loading={pendingJump === "latest"}
              onClick={() => setPendingJump("latest")}
            >
              <IconChevronDownOutline12 />
              Jump to latest
            </Button>
          )}
        </div>
      </div>
      <div
        ref={scrollRef}
        className="overflow-auto py-2 font-mono text-xs leading-5"
        style={{ height: fixedHeight }}
        onScroll={(event) => {
          const el = event.currentTarget;
          const scrolledUp = el.scrollTop < lastScrollTop.current;
          lastScrollTop.current = el.scrollTop;
          // Rows grow when they are measured, so only an upward scroll stops
          // following, not a list that grew below the viewport
          if (el.scrollHeight - el.scrollTop - el.clientHeight < TAIL_FOLLOW_SLACK_PX) {
            setFollowsTail(true);
          } else if (scrolledUp) {
            setFollowsTail(false);
          }
        }}
      >
        <div className="relative w-full" style={{ height: totalSize }}>
          {virtualizer.getVirtualItems().map((item) => (
            <div
              key={item.key}
              data-index={item.index}
              ref={virtualizer.measureElement}
              className="absolute left-0 top-0 w-full"
              style={{ transform: `translateY(${item.start}px)` }}
            >
              {item.index < lines.length ? (
                <BuildLogLineRow line={lines[item.index]} onExpandRun={expandRun} />
              ) : (
                <div className="flex h-6 items-center gap-2 px-4 text-gray-10">
                  <span className={GUTTER_WIDTH_CLASS} />
                  <Loading size={12} />
                  Loading more
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

const STEP_NAME_PATTERN = /^(\[[^\]]+\])\s*(.*)$/;
const LOG_TIME_FORMAT = new Intl.DateTimeFormat(undefined, {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  fractionalSecondDigits: 3,
  hourCycle: "h23",
});
const GUTTER_WIDTH_CLASS = "w-[12ch] shrink-0";

function BuildLogLineRow({
  line,
  onExpandRun,
}: {
  line: BuildLogLine;
  onExpandRun: (runKey: string) => void;
}) {
  switch (line.kind) {
    case "step": {
      const stepNameParts = STEP_NAME_PATTERN.exec(line.step);
      return (
        <div className="flex items-center gap-2 border-t border-grayA-3 bg-grayA-2 px-4 py-1.5">
          <IconChevronRightOutline12 className="size-3 shrink-0 text-gray-9" />
          <span className="min-w-0 truncate">
            {stepNameParts && <span className="text-gray-11">{stepNameParts[1]} </span>}
            <span className="text-gray-12">{stepNameParts ? stepNameParts[2] : line.step}</span>
          </span>
        </div>
      );
    }
    case "fold":
      return (
        <button
          type="button"
          onClick={() => onExpandRun(line.runKey)}
          className="group flex w-full cursor-pointer items-center gap-4 px-4 py-1 text-left text-gray-11 hover:bg-grayA-2 hover:text-gray-12 focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-gray-7"
        >
          <span className={GUTTER_WIDTH_CLASS} />
          <span className="flex items-center gap-1.5">
            <IconChevronExpandYOutline12 className="size-3 text-gray-9 group-hover:text-gray-11" />
            Show {line.entriesHidden.toLocaleString()} more entries
          </span>
        </button>
      );
    case "entry": {
      const isCached = line.text === "CACHED";
      const isDone = line.text.startsWith("DONE ");
      return (
        <div
          className={cn(
            "flex gap-4 px-4 py-0.5 hover:bg-grayA-2",
            line.tone === "error" &&
              "bg-error-2 hover:bg-error-2 dark:bg-error-3 dark:hover:bg-error-3",
          )}
        >
          <time
            dateTime={new Date(line.entry.time).toISOString()}
            title={new Date(line.entry.time).toLocaleString()}
            className={cn(GUTTER_WIDTH_CLASS, "tabular-nums text-grayA-10")}
          >
            {LOG_TIME_FORMAT.format(line.entry.time)}
          </time>
          <span
            className={cn(
              "min-w-0 whitespace-pre-wrap break-all",
              isCached || isDone
                ? "inline-flex items-center gap-1.5 text-gray-11"
                : {
                    stdout: "text-gray-12",
                    stderr: "text-warning-11",
                    error: "text-error-11",
                  }[line.tone],
            )}
          >
            {isCached && <IconBoltOutline18 className="size-3 shrink-0 text-gray-9" />}
            {isDone && <IconCheckOutline12 className="size-3 shrink-0 text-success-11" />}
            {line.text}
          </span>
        </div>
      );
    }
  }
}

function BuildLogsMessage({ children }: { children: ReactNode }) {
  return <div className="flex items-center gap-2 px-4 py-6 text-xs text-gray-11">{children}</div>;
}
