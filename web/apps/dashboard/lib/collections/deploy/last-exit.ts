import type { LastExit } from "@/lib/types/deploy";
import { isFatalWaitingReason } from "./instance-status";

export type ExitSummary =
  | { kind: "waiting"; reason: string }
  | { kind: "exit"; reason: string | null; exit: string | null };

export function describeLastExit(lastExit: LastExit): ExitSummary {
  if (isFatalWaitingReason(lastExit.statusReason)) {
    return { kind: "waiting", reason: lastExit.statusReason };
  }
  const { exitCode } = lastExit;
  return {
    kind: "exit",
    reason: lastExit.reason,
    exit: exitCode !== null && exitCode !== 0 ? `exit=${exitCode}` : null,
  };
}

export function latestExitAcross(
  instances: readonly { lastExit: LastExit | null }[],
): LastExit | null {
  return instances.reduce<LastExit | null>((latest, { lastExit }) => {
    if (!lastExit) {
      return latest;
    }
    if (!latest) {
      return lastExit;
    }
    const at = lastExit.finishedAt ?? -1;
    const latestAt = latest.finishedAt ?? -1;
    return at > latestAt || (at === latestAt && lastExit.restartCount > latest.restartCount)
      ? lastExit
      : latest;
  }, null);
}
