"use client";

import { isFatalWaitingReason, reasonCopy } from "@/lib/collections/deploy/exit-reasons";
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

type ExitKind = "waiting" | "exit";

type ExitCopy = {
  kind: ExitKind;
  label: string;
  exit: string | null;
  body: string | null;
  hint: { label: string; body: string } | null;
  raw: string | null;
};

function rawOf(lastExit: LastExit): string | null {
  const reasons = [...new Set([lastExit.statusReason, lastExit.reason])];
  const code = lastExit.exitCode !== null ? `exit code ${lastExit.exitCode}` : null;
  return [...reasons, code].filter(Boolean).join(" · ") || null;
}

function exitCopy(lastExit: LastExit): ExitCopy {
  const waiting = isFatalWaitingReason(lastExit.statusReason);
  const reason = waiting ? lastExit.statusReason : lastExit.reason;
  const copy = reasonCopy(reason);
  const failedCode =
    !waiting && lastExit.exitCode !== null && lastExit.exitCode !== 0 ? lastExit.exitCode : null;
  const generic = copy === null || reason === "Error";
  return {
    kind: waiting ? "waiting" : "exit",
    label: copy?.label ?? reason ?? "Exited",
    exit: failedCode !== null ? `exit=${failedCode}` : null,
    body: copy?.body ?? null,
    hint: generic && failedCode !== null ? describeExitCode(failedCode, lastExit.signal) : null,
    raw: rawOf(lastExit),
  };
}

const TEXT_TONE: Record<ExitKind, string> = {
  waiting: "text-error-11",
  exit: "text-warning-11",
};

const BANNER_VARIANT: Record<ExitKind, "error" | "warning"> = {
  waiting: "error",
  exit: "warning",
};

export function LastExitText({ lastExit }: { lastExit: LastExit }) {
  const copy = exitCopy(lastExit);
  const lines = [{ label: copy.label, body: copy.body }, ...(copy.hint ? [copy.hint] : [])];
  return (
    <InfoTooltip
      content={
        <div className="flex max-w-[280px] flex-col gap-1.5">
          {lines.map((line) => (
            <div key={line.label} className="flex flex-col gap-0.5">
              <div className="font-medium">{line.label}</div>
              {line.body && <div>{line.body}</div>}
            </div>
          ))}
          {copy.raw && <div className="font-mono opacity-70">{copy.raw}</div>}
        </div>
      }
      position={{ side: "top", align: "end" }}
      triggerClassName={cn(
        "rounded-sm text-xs font-medium whitespace-nowrap",
        TEXT_TONE[copy.kind],
      )}
    >
      {copy.label}
      {copy.exit && <span className="ml-1 font-mono tabular-nums">· {copy.exit}</span>}
    </InfoTooltip>
  );
}

export function LastExitCard({ lastExit }: { lastExit: LastExit }) {
  const copy = exitCopy(lastExit);
  const description = [copy.hint?.body, copy.body].filter(Boolean).join(" ");
  return (
    <AlertBanner variant={BANNER_VARIANT[copy.kind]} appearance="card">
      <IconTriangleWarningOutline18 className="size-4" />
      <AlertBannerTitle>
        {copy.raw ? (
          <InfoTooltip
            content={<span className="font-mono">{copy.raw}</span>}
            position={{ side: "top", align: "start" }}
            triggerClassName="rounded-sm text-left"
          >
            {copy.label}
          </InfoTooltip>
        ) : (
          copy.label
        )}
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
      {description && <AlertBannerDescription>{description}</AlertBannerDescription>}
    </AlertBanner>
  );
}

// describeExitCode maps the exit codes users actually see to plain-
// language hints. Codes ≥128 conventionally mean "force-killed", but
// users don't need the signal arithmetic — just whether it's a memory
// issue or an app crash and what to look at next.
const EXIT_CODE_DESCRIPTIONS: Record<number, { label: string; body: string }> = {
  1: {
    label: "exit=1",
    body: "Unhandled error or failed startup check.",
  },
  2: {
    label: "exit=2",
    body: "Crashed, usually a panic or uncaught exception.",
  },
  126: {
    label: "exit=126",
    body: "Start command found but not executable.",
  },
  127: {
    label: "exit=127",
    body: "Start command not found.",
  },
  128: {
    label: "exit=128",
    body: "Force-killed, usually out of memory.",
  },
  137: {
    label: "exit=137",
    body: "Force-killed, usually out of memory.",
  },
  139: {
    label: "exit=139",
    body: "Invalid memory access in your code.",
  },
  143: {
    label: "exit=143",
    body: "Shut down on request. Normal during a redeploy.",
  },
};

function describeExitCode(
  exitCode: number,
  signal: number | null,
): { label: string; body: string } | null {
  const description = EXIT_CODE_DESCRIPTIONS[exitCode];
  if (description) {
    return description;
  }

  if (signal && signal > 0) {
    return {
      label: `exit=${exitCode}`,
      body: "Force-killed, usually out of memory.",
    };
  }

  return {
    label: `exit=${exitCode}`,
    body: "See your app's docs for this exit code.",
  };
}
