import { plural } from "@/lib/fmt";
import type { LastExit } from "@/lib/types/deploy";
import type { ContainerStatus } from "@unkey/db/src/schema";
import { type FatalWaitingReason, isFatalWaitingReason } from "./instance-status";

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
    reason: lastExit.reason || "Error",
    exit: exitCode !== null && exitCode !== 0 ? `exit=${exitCode}` : null,
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

// The badge's reason and exit code, then the restart count
export function exitSummary(lastExit: LastExit): string {
  const badge = exitBadge(lastExit);
  const cause = badge.kind === "waiting" ? [badge.reason] : [badge.reason, badge.exit];
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
