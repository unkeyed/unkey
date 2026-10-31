import {
  type FatalWaitingReason,
  isFatalWaitingReason,
  toInstanceKey,
} from "@/lib/collections/deploy/instance-status";
import type { InstanceEvent, InstanceEventKind } from "@/lib/metrics/app-metrics";
import type {
  InstanceEventKind as RawKind,
  CrashTimelineEvent as RawRow,
} from "@unkey/clickhouse/src/instance-events";

export type { RawRow };

const TERMINATED_REASONS = new Map<string, InstanceEventKind>([
  ["OOMKilled", "oom"],
  ["Completed", "completed"],
]);

const WAITING_KIND: Record<FatalWaitingReason, InstanceEventKind> = {
  CrashLoopBackOff: "crashloop",
  ImagePullBackOff: "image-pull",
  ErrImagePull: "image-pull",
  InvalidImageName: "image-pull",
  CreateContainerConfigError: "config",
};

export const EPISODE_WAITING_REASONS = Object.keys(WAITING_KIND);

const KIND_OF: Record<RawKind, (row: RawRow) => InstanceEventKind | null> = {
  running: () => "started",
  terminated: (row) =>
    TERMINATED_REASONS.get(row.reason) ?? (row.exit_code === 0 ? "completed" : "crashed"),
  waiting: (row) => (isFatalWaitingReason(row.reason) ? WAITING_KIND[row.reason] : null),
};

export function toInstanceEvents(
  rows: ReadonlyArray<RawRow>,
  instanceIds: ReadonlyMap<string, string>,
): InstanceEvent[] {
  return rows.flatMap((row) => {
    const kind = KIND_OF[row.event_kind](row);
    if (kind === null) {
      return [];
    }
    return [
      {
        id: `${row.time}:${row.pod_name}:${row.event_fingerprint}`,
        atMs: row.time,
        instance: row.pod_name,
        instanceId: instanceIds.get(toInstanceKey(row.pod_name, row.region)) ?? null,
        region: row.region,
        deploymentId: row.deployment_id,
        kind,
        reason: row.reason,
        exitCode: row.event_kind === "terminated" ? row.exit_code : null,
        message: row.message.trim() || null,
      },
    ];
  });
}
