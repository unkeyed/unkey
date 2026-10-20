import type { LastExit } from "@/lib/types/deploy";

export function describeLastExit(lastExit: LastExit): {
  reason: string | null;
  exit: string | null;
} {
  if (lastExit.statusReason === "CrashLoopBackOff") {
    return { reason: "CrashLoopBackOff", exit: null };
  }
  const { exitCode } = lastExit;
  return {
    reason: lastExit.reason,
    exit: exitCode !== null && exitCode !== 0 ? `exit=${exitCode}` : null,
  };
}
