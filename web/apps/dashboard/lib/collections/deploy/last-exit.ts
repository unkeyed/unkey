import { plural } from "@/lib/fmt";
import type { LastExit } from "@/lib/types/deploy";
import type { ContainerStatus } from "@unkey/db/src/schema";
import { match } from "@unkey/match";
import { type FatalWaitingReason, isFatalWaitingReason } from "./instance-status";

function exitCodeLabel(exitCode: number): string {
  return `exit=${exitCode}`;
}

export type ExitBadge =
  | { kind: "waiting"; reason: FatalWaitingReason }
  | { kind: "exit"; reason: string; exit: string | null };

export function exitBadge(lastExit: LastExit): ExitBadge {
  if (isFatalWaitingReason(lastExit.statusReason)) {
    return { kind: "waiting", reason: lastExit.statusReason };
  }
  const { exitCode } = lastExit;
  return {
    kind: "exit",
    reason: lastExit.reason || (exitCode === 0 ? "Completed" : "Error"),
    exit: exitCode !== null && exitCode !== 0 ? exitCodeLabel(exitCode) : null,
  };
}

export function lastExitOf(containerStatus: ContainerStatus | null): LastExit | null {
  const term = containerStatus?.lastTerminationState ?? null;
  const waitingReason = containerStatus?.waiting?.reason ?? null;
  const statusReason = isFatalWaitingReason(waitingReason) ? waitingReason : null;
  if (term === null && statusReason === null) {
    return null;
  }
  return {
    restartCount: containerStatus?.restartCount ?? 0,
    exitCode: term?.exitCode ?? null,
    signal: term?.signal ?? null,
    reason: term?.reason ?? null,
    finishedAt: term?.finishedAt ?? null,
    statusReason,
  };
}

export function exitSummary(lastExit: LastExit): string {
  const badge = exitBadge(lastExit);
  const cause = match(badge)
    .with({ kind: "waiting" }, ({ reason }) => [reason])
    .with({ kind: "exit" }, ({ reason, exit }) => [reason, exit])
    .exhaustive();
  const restarts = lastExit.restartCount > 0 ? plural(lastExit.restartCount, "restart") : null;
  return [...cause, restarts].filter(Boolean).join(" · ");
}

// latestExit picks one exit across all instances of a deployment so that a
// multi-region rollout with one failing pod still surfaces the failure even
// when others are healthy. A pod stuck on a fatal waiting reason outranks any
// pod that recovered, however recent its exit; after that the latest
// finishedAt wins, then the higher restartCount.
export function latestExit(exits: readonly (LastExit | null)[]): LastExit | null {
  let result: LastExit | null = null;
  for (const candidate of exits) {
    if (candidate && (!result || outranks(candidate, result))) {
      result = candidate;
    }
  }
  return result;
}

function outranks(a: LastExit, b: LastExit): boolean {
  const fatalA = isFatalWaitingReason(a.statusReason);
  const fatalB = isFatalWaitingReason(b.statusReason);
  if (fatalA !== fatalB) {
    return fatalA;
  }
  const atA = a.finishedAt ?? -1;
  const atB = b.finishedAt ?? -1;
  return atA > atB || (atA === atB && a.restartCount > b.restartCount);
}

export function exitDetail(lastExit: LastExit): string | null {
  const { reason, exitCode } = lastExit;
  const parts = match(exitBadge(lastExit))
    .with({ kind: "waiting" }, () => [reason, exitCode !== null ? exitCodeLabel(exitCode) : null])
    .with({ kind: "exit" }, () => [exitCode === 0 ? exitCodeLabel(0) : null])
    .exhaustive();
  return parts.filter(Boolean).join(" · ") || null;
}
