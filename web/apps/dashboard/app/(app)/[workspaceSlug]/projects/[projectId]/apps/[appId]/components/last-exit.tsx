"use client";

import { type ExitSummary, summarizeExit } from "@/lib/collections/deploy/exit-reasons";
import { plural } from "@/lib/fmt";
import type { LastExit } from "@/lib/types/deploy";
import { IconTriangleWarningOutline18 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerDescription,
  AlertBannerMeta,
  AlertBannerTitle,
  InfoTooltip,
  TimestampInfo,
} from "@unkey/ui";
import { cn } from "cn";
import type { ReactNode } from "react";

const TEXT_TONE: Record<ExitSummary["severity"], string> = {
  error: "text-error-11",
  warning: "text-warning-11",
};

function ExitDetails({ summary }: { summary: ExitSummary }) {
  return (
    <div className="flex max-w-[280px] flex-col gap-1">
      <div className="font-medium">{summary.label}</div>
      {summary.sentences.map((sentence) => (
        <div key={sentence}>{sentence}</div>
      ))}
      {summary.raw && <div className="font-mono opacity-70">{summary.raw}</div>}
    </div>
  );
}

export function LastExitText({ summary }: { summary: ExitSummary }) {
  return (
    <InfoTooltip
      content={<ExitDetails summary={summary} />}
      position={{ side: "top", align: "end" }}
      triggerClassName={cn(
        "rounded-sm text-xs font-medium whitespace-nowrap",
        TEXT_TONE[summary.severity],
      )}
    >
      {summary.label}
      {summary.exit && <span className="ml-1 font-mono tabular-nums">· {summary.exit}</span>}
    </InfoTooltip>
  );
}

export function LastExitHover({
  summary,
  children,
}: { summary: ExitSummary; children: ReactNode }) {
  return (
    <InfoTooltip
      content={<ExitDetails summary={summary} />}
      position={{ side: "top", align: "start" }}
      triggerClassName="relative z-20 min-w-0 truncate rounded-sm underline decoration-grayA-6 decoration-dotted underline-offset-4"
    >
      {children}
    </InfoTooltip>
  );
}

export function LastExitCard({ lastExit }: { lastExit: LastExit }) {
  const summary = summarizeExit(lastExit);
  return (
    <AlertBanner variant={summary.severity} appearance="card">
      <IconTriangleWarningOutline18 className="size-4" />
      <AlertBannerTitle>
        <RawReason label={summary.label} raw={summary.raw} />
      </AlertBannerTitle>
      <AlertBannerMeta className="flex items-center gap-1 tabular-nums">
        {lastExit.restartCount > 0 && <span>{plural(lastExit.restartCount, "restart")}</span>}
        {lastExit.restartCount > 0 && lastExit.finishedAt && <span aria-hidden="true">·</span>}
        {lastExit.finishedAt && (
          <TimestampInfo
            value={lastExit.finishedAt}
            displayType="relative"
            className="text-gray-10"
          />
        )}
      </AlertBannerMeta>
      {summary.sentences.length > 0 && (
        <AlertBannerDescription>{summary.sentences.join(" ")}</AlertBannerDescription>
      )}
    </AlertBanner>
  );
}

function RawReason({ label, raw }: { label: string; raw: string | null }) {
  if (!raw) {
    return label;
  }
  return (
    <InfoTooltip
      content={<span className="font-mono">{raw}</span>}
      position={{ side: "top", align: "start" }}
      triggerClassName="rounded-sm text-left"
    >
      {label}
    </InfoTooltip>
  );
}
