"use client";

import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
  IconBoltOutline18,
  IconCheckOutline12,
  IconChevronExpandYOutline12,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Loading } from "@unkey/ui";
import { cn } from "cn";
import { type ReactNode, useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
import { useDeployment } from "../../layout-provider";
import { BUILD_LOG_ENTRIES_SHOWN_MAX, useBuildLogs } from "../../use-build-logs";
import { type BuildLogLine, type Tone, toBuildLogLines, toBuildLogText } from "./build-log-lines";
import { BuildLogsHeader } from "./build-logs-header";
import {
  type JumpCursor,
  type JumpTone,
  type PendingJump,
  type TailRow,
  buildLogsHeaderState,
  scrollToLatest,
  tailRow,
  toToneIndices,
} from "./build-logs-header-state";

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
  isOpen?: boolean;
};

export function DeploymentBuildLogs({
  fixedHeight = 500,
  focusErrorTick = 0,
  isOpen = true,
}: Props) {
  const { deployment } = useDeployment();
  const isFailed = deployment.status === "failed";
  const isBuilding = BUILD_NOT_DONE_STATUSES.has(deployment.status);
  const [followsTail, setFollowsTail] = useState(isBuilding);
  const [pendingJump, setPendingJump] = useState<PendingJump | null>(
    isFailed ? "last-error" : null,
  );
  const [jumpCursor, setJumpCursor] = useState<JumpCursor | null>(null);
  const [query, setQuery] = useState("");
  const shownQuery = useDeferredValue(query);
  const searchRef = useRef<HTMLInputElement>(null);
  // Rows near a jump target are not measured yet, so the jump repeats as they
  // are, until the viewer scrolls
  const [jumpTargetIndex, setJumpTargetIndex] = useState<number | null>(null);
  const logHeight = `min(${fixedHeight}px, 60dvh)`;
  const readsToEnd = isFailed || followsTail || pendingJump !== null || query.trim() !== "";

  const logs = useBuildLogs(deployment, { readsToEnd });
  const entries = logs.data?.entries;
  const hasMore = logs.data?.hasMore === true;
  const isCaughtUp = logs.data !== undefined && !logs.data.hasMore;

  const [expandedRunKeys, setExpandedRunKeys] = useState<ReadonlySet<string>>(new Set());
  const lines = useMemo(
    () => toBuildLogLines(entries ?? [], expandedRunKeys, shownQuery),
    [entries, expandedRunKeys, shownQuery],
  );
  const toneIndices = useMemo(() => toToneIndices(lines), [lines]);
  const tail = tailRow({ hasMore, isBuilding });

  const scrollRef = useRef<HTMLDivElement>(null);
  const lastScrollTop = useRef(0);
  const virtualizer = useVirtualizer({
    count: tail.type === "none" ? lines.length : lines.length + 1,
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

  const jumpTo = (tone: JumpTone, position: number) => {
    const index = toneIndices[tone].at(position);
    if (index === undefined) {
      return;
    }
    setFollowsTail(false);
    setJumpCursor({
      tone,
      position: (position + toneIndices[tone].length) % toneIndices[tone].length,
    });
    setJumpTargetIndex(index);
  };

  // Rows measured during the smooth scroll grow the list, so following takes
  // over at its end and scrolls the rest of the way
  const scrollToEnd = () => {
    const el = scrollRef.current;
    if (!el) {
      return;
    }
    setJumpTargetIndex(null);
    setJumpCursor(null);
    el.addEventListener("scrollend", () => setFollowsTail(true), { once: true });
    el.scrollTo({
      top: el.scrollHeight,
      behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth",
    });
  };

  const stepTone = (tone: JumpTone, delta: 1 | -1) => {
    const count = toneIndices[tone].length;
    if (jumpCursor?.tone === tone && count > 0) {
      jumpTo(tone, (jumpCursor.position + delta + count) % count);
    }
  };

  const toggleTone = (tone: JumpTone) => {
    setJumpTargetIndex(null);
    if (jumpCursor?.tone === tone) {
      setJumpCursor(null);
    } else if (isCaughtUp) {
      jumpTo(tone, 0);
    } else {
      setPendingJump(tone);
    }
  };

  // biome-ignore lint/correctness/useExhaustiveDependencies: jumpTo reads the same state as the listed values
  useEffect(() => {
    if (pendingJump === null || !isCaughtUp) {
      return;
    }
    setPendingJump(null);
    if (pendingJump === "latest") {
      scrollToEnd();
    } else if (pendingJump === "last-error") {
      jumpTo("error", -1);
    } else {
      jumpTo(pendingJump, 0);
    }
  }, [pendingJump, isCaughtUp, toneIndices]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: totalSize is the trigger, it changes as rows are measured
  useEffect(() => {
    if (jumpTargetIndex !== null) {
      virtualizer.scrollToIndex(jumpTargetIndex, { align: "center" });
    }
  }, [jumpTargetIndex, totalSize, virtualizer]);

  useEffect(() => {
    if (focusErrorTick > 0) {
      setPendingJump("last-error");
    }
  }, [focusErrorTick]);

  // A new query renumbers the lines, so old jump targets point at other lines
  const changeQuery = (nextQuery: string) => {
    setQuery(nextQuery);
    setJumpTargetIndex(null);
    setJumpCursor(null);
  };

  const focusSearch = () => {
    searchRef.current?.focus();
    searchRef.current?.select();
  };
  // A folded card keeps the logs mounted, so native find stays in charge until
  // the logs are on screen
  const [isMac] = useState(
    () => typeof navigator !== "undefined" && navigator.userAgent.includes("Mac"),
  );
  const searchShortcutDisabled = !isOpen || !entries || entries.length === 0;
  useKeyboardShortcut("meta+f", focusSearch, {
    ignoreInputs: false,
    disabled: searchShortcutDisabled || !isMac,
  });
  useKeyboardShortcut("ctrl+f", focusSearch, {
    ignoreInputs: false,
    disabled: searchShortcutDisabled || isMac,
  });

  const expandRun = (runKey: string) => {
    setFollowsTail(false);
    setJumpTargetIndex(null);
    setExpandedRunKeys((previous) => new Set(previous).add(runKey));
  };

  if (logs.isLoading) {
    return (
      <div className="flex flex-col">
        <div className="flex h-11 items-center gap-2 border-b border-grayA-3 px-4 text-xs text-gray-10">
          <Loading size={12} />
          Loading build logs
        </div>
        <div style={{ height: logHeight }} />
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
  if (!entries || entries.length === 0) {
    return <BuildLogsMessage>No build logs yet</BuildLogsMessage>;
  }

  const header = buildLogsHeaderState({
    toneIndices,
    cursor: jumpCursor,
    pendingJump,
    hasMore,
    isFailed,
    entriesTotal: logs.data.entriesTotal,
    entriesShownMax: BUILD_LOG_ENTRIES_SHOWN_MAX,
  });

  return (
    <div className="flex flex-col">
      <BuildLogsHeader
        label={header.label}
        counters={header.counters}
        copyText={() =>
          toBuildLogText(
            jumpCursor
              ? toneIndices[jumpCursor.tone].map((index) => lines[index])
              : toBuildLogLines(entries, "all"),
          )
        }
        scrollToLatest={scrollToLatest({ followsTail, hasMore, isAtBottom, pendingJump })}
        onScrollToLatest={() => (isCaughtUp ? scrollToEnd() : setPendingJump("latest"))}
        onToggleTone={toggleTone}
        onStepTone={stepTone}
        query={query}
        onQueryChange={changeQuery}
        searchRef={searchRef}
        searchShortcutLabel={isMac ? "⌘F" : "Ctrl F"}
      />
      <div
        ref={scrollRef}
        className="overflow-auto py-2 font-mono text-xs leading-5"
        style={{ height: logHeight }}
        onWheel={() => setJumpTargetIndex(null)}
        onTouchMove={() => setJumpTargetIndex(null)}
        onKeyDown={() => setJumpTargetIndex(null)}
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
        {lines.length === 0 && (
          <div className="px-4 py-4 text-gray-11">No lines match "{shownQuery.trim()}"</div>
        )}
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
                <BuildLogLineRow
                  line={lines[item.index]}
                  query={shownQuery}
                  onExpandRun={expandRun}
                />
              ) : (
                <TailLine tail={tail} />
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

const LOG_TIME_FORMAT = new Intl.DateTimeFormat(undefined, {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  fractionalSecondDigits: 3,
  hourCycle: "h23",
});
const GUTTER_WIDTH_CLASS = "w-[12ch] shrink-0";

const ENTRY_TONE_CLASSES: Record<Tone, { row: string; text: string }> = {
  stdout: { row: "", text: "text-gray-12" },
  stderr: { row: "", text: "text-gray-12" },
  warning: {
    row: "bg-warning-2 hover:bg-warning-2 dark:bg-warning-3 dark:hover:bg-warning-3",
    text: "text-warning-11",
  },
  error: {
    row: "bg-error-2 hover:bg-error-2 dark:bg-error-3 dark:hover:bg-error-3",
    text: "text-error-11",
  },
  event: { row: "", text: "inline-flex items-center gap-1.5 text-gray-11" },
};

function TailLine({ tail }: { tail: TailRow }) {
  return match(tail)
    .with({ type: "none" }, () => null)
    .with({ type: "loading-more" }, () => (
      <div className="flex gap-4 px-4 py-0.5 text-gray-10">
        <span className={GUTTER_WIDTH_CLASS} />
        <span className="flex items-center gap-2">
          <Loading size={12} />
          Loading more
        </span>
      </div>
    ))
    .with({ type: "running" }, () => (
      <div className="flex gap-4 px-4 py-0.5 text-info-11">
        <span className={cn(GUTTER_WIDTH_CLASS, "flex items-center justify-end pr-1")}>
          <span className="size-1.5 animate-pulse rounded-full bg-current" />
        </span>
        Running
      </div>
    ))
    .exhaustive();
}

function BuildLogLineRow({
  line,
  query,
  onExpandRun,
}: {
  line: BuildLogLine;
  query: string;
  onExpandRun: (runKey: string) => void;
}) {
  return match(line)
    .with({ kind: "step" }, ({ step }) => (
      <div className="px-4 py-0.5 font-semibold text-gray-12">{step}</div>
    ))
    .with({ kind: "fold" }, ({ runKey, entriesHidden }) => (
      <button
        type="button"
        onClick={() => onExpandRun(runKey)}
        className="group flex w-full cursor-pointer items-center gap-4 px-4 py-1 text-left text-gray-11 hover:bg-grayA-2 hover:text-gray-12 focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-gray-7"
      >
        <span className={GUTTER_WIDTH_CLASS} />
        <span className="flex items-center gap-1.5">
          <IconChevronExpandYOutline12 className="size-3 text-gray-9 group-hover:text-gray-11" />
          Show {entriesHidden.toLocaleString()} more lines
        </span>
      </button>
    ))
    .with({ kind: "entry" }, ({ entry, tone, text }) => (
      <div className={cn("flex gap-4 px-4 py-0.5 hover:bg-grayA-2", ENTRY_TONE_CLASSES[tone].row)}>
        <time
          dateTime={new Date(entry.time).toISOString()}
          title={new Date(entry.time).toLocaleString()}
          className={cn(GUTTER_WIDTH_CLASS, "tabular-nums text-grayA-10")}
        >
          {LOG_TIME_FORMAT.format(entry.time)}
        </time>
        <span
          className={cn("min-w-0 whitespace-pre-wrap break-all", ENTRY_TONE_CLASSES[tone].text)}
        >
          {tone === "event" &&
            (text === "CACHED" ? (
              <IconBoltOutline18 className="size-3 shrink-0 text-gray-9" />
            ) : (
              <IconCheckOutline12 className="size-3 shrink-0 text-success-11" />
            ))}
          <LogText text={text} query={query} />
        </span>
      </div>
    ))
    .exhaustive();
}

const URL_PATTERN = /https?:\/\/[^\s"'<>`]+[^\s"'<>`.,:;)\]]/g;

function LogText({ text, query }: { text: string; query: string }) {
  const parts: ReactNode[] = [];
  let offset = 0;
  for (const url of text.matchAll(URL_PATTERN)) {
    parts.push(
      <Highlighted key={`text-${offset}`} text={text.slice(offset, url.index)} query={query} />,
    );
    parts.push(
      <a
        key={`url-${url.index}`}
        href={url[0]}
        target="_blank"
        rel="noreferrer"
        className="underline decoration-grayA-8 underline-offset-2 hover:decoration-current"
      >
        <Highlighted text={url[0]} query={query} />
      </a>,
    );
    offset = url.index + url[0].length;
  }
  parts.push(<Highlighted key={`text-${offset}`} text={text.slice(offset)} query={query} />);
  return parts;
}

function Highlighted({ text, query }: { text: string; query: string }) {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return text;
  }
  const parts: ReactNode[] = [];
  const haystack = text.toLowerCase();
  let offset = 0;
  for (
    let found = haystack.indexOf(needle);
    found !== -1;
    found = haystack.indexOf(needle, offset)
  ) {
    parts.push(text.slice(offset, found));
    parts.push(
      <mark key={found} className="rounded-xs bg-warning-5 text-gray-12">
        {text.slice(found, found + needle.length)}
      </mark>,
    );
    offset = found + needle.length;
  }
  parts.push(text.slice(offset));
  return parts;
}

function BuildLogsMessage({ children }: { children: ReactNode }) {
  return <div className="flex items-center gap-2 px-4 py-6 text-xs text-gray-11">{children}</div>;
}
