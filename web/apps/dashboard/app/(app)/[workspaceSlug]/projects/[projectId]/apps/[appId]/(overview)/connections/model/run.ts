import { CALLER_KINDS } from "./caller-kind";
import type { ConnectionOp } from "./plan";

export type RunResult = {
  applied: readonly ConnectionOp[];
  failure: { op: ConnectionOp; error: unknown } | null;
};

export async function runInOrder<R>(
  ops: readonly ConnectionOp[],
  apply: (op: ConnectionOp) => Promise<R>,
): Promise<RunResult> {
  const applied: ConnectionOp[] = [];
  for (const op of ops) {
    try {
      await apply(op);
    } catch (error) {
      return { applied, failure: { op, error } };
    }
    applied.push(op);
  }
  return { applied, failure: null };
}

export function redeployEnvironmentIds(applied: readonly ConnectionOp[]): string[] {
  const ids = applied
    .filter((op) => CALLER_KINDS[op.environment.kind].needsRedeploy)
    .map((op) => op.environment.id);
  return [...new Set(ids)];
}

export function needsRedeploy(applied: readonly ConnectionOp[]): boolean {
  return redeployEnvironmentIds(applied).length > 0;
}

export function failureDescription(
  applied: readonly ConnectionOp[],
  failed: ConnectionOp,
  verb: string,
  message: string,
): string {
  if (applied.length === 0) {
    return message;
  }
  const done = applied.map((op) => CALLER_KINDS[op.environment.kind].label).join(" and ");
  return `${done} ${verb}. ${CALLER_KINDS[failed.environment.kind].label} failed: ${message}`;
}
