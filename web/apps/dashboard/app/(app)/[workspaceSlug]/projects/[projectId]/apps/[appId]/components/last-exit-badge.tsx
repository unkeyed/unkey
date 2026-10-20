"use client";

import { describeLastExit } from "@/lib/collections/deploy/last-exit";
import type { LastExit } from "@/lib/types/deploy";
import { match } from "@unkey/match";
import { Badge, InfoTooltip } from "@unkey/ui";

export function LastExitBadge({ lastExit }: { lastExit: LastExit }) {
  const isCrashloop = lastExit.statusReason === "CrashLoopBackOff";
  const summary = describeLastExit(lastExit);
  const reason = summary.reason ?? "Error";
  const variant = isCrashloop ? "warning" : "error";

  const tooltip = explainExit(reason, lastExit.exitCode, lastExit.signal);

  return (
    <InfoTooltip content={tooltip} position={{ side: "top", align: "end" }}>
      <Badge variant={variant} className="text-xs whitespace-nowrap">
        {reason}
        {summary.exit !== null ? (
          <span className="ml-1 font-mono tabular-nums">· {summary.exit}</span>
        ) : null}
      </Badge>
    </InfoTooltip>
  );
}

// explainExit produces tooltip for LastExitBadge with user-friendly descriptions.
function explainExit(
  reason: string,
  exitCode: number | null,
  signal: number | null,
): React.ReactNode {
  // CrashLoopBackOff: point users at the exit code for diagnosis.
  if (reason === "CrashLoopBackOff") {
    return (
      <div className="flex flex-col gap-1.5 max-w-[280px]">
        <div className="font-medium">App keeps crashing on startup</div>
        <div>
          Your app has exited too many times in a row, so we're slowing down restart attempts to
          give it room to recover.
        </div>
        <div>
          Check the recent crash entries below for the exit code that's causing the loop, and your
          logs for the underlying error.
        </div>
      </div>
    );
  }

  const lines: { label: string; body: string }[] = [];

  const reasonLine = match(reason)
    .with("OOMKilled", () => ({
      label: "Out of memory",
      body: "Your app used more memory than its configured limit. Either it has a memory leak, or the limit is set too low for what it actually needs at peak.",
    }))
    .with("Error", () => ({
      label: "App exited with an error",
      body: "Your app stopped with a non-zero status. The exit code below narrows down how.",
    }))
    .with("ContainerCannotRun", () => ({
      label: "Couldn't start your app",
      body: "We were unable to start your app at all. Usually means the start command, entrypoint, or image is broken. Check the command and build settings of your app.",
    }))
    .with("Completed", () => ({
      label: "Exited cleanly",
      body: "Your app shut down without an error. Normal for one-off jobs; unusual for a service that's supposed to keep running. Check whether your main loop returned early.",
    }))
    .when(
      (r) => Boolean(r),
      (r) => ({
        label: r,
        body: "Your app exited. The exit code below has more detail.",
      }),
    )
    .otherwise(() => null);

  if (reasonLine) {
    lines.push(reasonLine);
  }

  if (exitCode !== null && exitCode !== 0) {
    const exitLine = describeExitCode(exitCode, signal);
    if (exitLine) {
      lines.push(exitLine);
    }
  }

  if (lines.length === 0) {
    return reason || "App exited";
  }

  return (
    <div className="flex flex-col gap-1.5 max-w-[280px]">
      {lines.map((line) => (
        <div key={line.label} className="flex flex-col gap-0.5">
          <div className="font-medium">{line.label}</div>
          <div>{line.body}</div>
        </div>
      ))}
    </div>
  );
}

// describeExitCode maps the exit codes users actually see to plain-
// language hints. Codes ≥128 conventionally mean "force-killed", but
// users don't need the signal arithmetic — just whether it's a memory
// issue or an app crash and what to look at next.
const EXIT_CODE_DESCRIPTIONS: Record<number, { label: string; body: string }> = {
  1: {
    label: "exit=1",
    body: "Your app returned an error. Check your logs for the cause. Typically an unhandled error or a failed startup check.",
  },
  2: {
    label: "exit=2",
    body: "Your app crashed with an unhandled error. Usually a panic or uncaught exception. Your logs should have a stack trace.",
  },
  126: {
    label: "exit=126",
    body: "We found your start command but couldn't run it. Most often a permissions issue or wrong shebang. Make sure the binary is executable.",
  },
  127: {
    label: "exit=127",
    body: "We couldn't find the command you asked us to run. Check that your start command and binary path are correct in your build settings.",
  },
  128: {
    label: "exit=128",
    body: "Your app was force-killed. Often a memory issue (same root cause as exit 137, see that one for more detail).",
  },
  137: {
    label: "exit=137",
    body: "Your app was force-killed, almost always because it ran out of memory. Either it has a leak, or you need to raise its memory limit.",
  },
  139: {
    label: "exit=139",
    body: "Your app crashed due to an invalid memory access. This is a bug in your code, usually a null pointer or buffer overrun. Check your stack trace.",
  },
  143: {
    label: "exit=143",
    body: "Your app shut down cleanly when asked to. Normal during a redeploy; unusual otherwise.",
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
      body: "Your app was force-killed. Often a memory issue. Check your app's memory usage near the time of the crash.",
    };
  }

  return {
    label: `exit=${exitCode}`,
    body: "Your app exited with this code. Check your app's docs for what it means. Most apps document their exit codes.",
  };
}
