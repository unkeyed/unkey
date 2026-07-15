import { z } from "zod";
import type { Querier } from "./client/interface";

const TABLE = "default.instance_events_raw_v1";

// Mirrors the proto's InstanceEvent.state oneof case names. ctrl writes one
// of these strings into the CH event_kind column. Exported so trpc routers
// and dashboard components can import the same canonical set.
export const instanceEventKind = z.enum(["running", "terminated", "waiting"]);
export type InstanceEventKind = z.infer<typeof instanceEventKind>;

export const instanceEventsRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  // Environment + deployment are optional so the same query powers both
  // the deployment-scoped panel (everything set) and the project-wide
  // runtime-logs timeline (project only). The CH WHERE clause skips
  // unset filters via the Nullable(...) pattern.
  environmentId: z.string().nullable(),
  deploymentId: z.string().nullable(),
  // Optional time bounds. Logs viewer passes a tight range; the events
  // panel pages over the full TTL window.
  startTime: z.int().nullable(),
  endTime: z.int().nullable(),
  // Optional pod_uid filter for the logs viewer enrichment ("show events
  // for the same pod the user is reading logs from"). Empty array = all pods.
  podUids: z.array(z.string()),
  // Optional pod name filter — the runtime-logs page exposes the user-
  // facing instance ID which corresponds to k8s pod name in events.
  // Empty array = all pods.
  podNames: z.array(z.string()),
  // Optional region filter — mirrors the same filter on the logs query.
  // Empty array = all regions.
  regions: z.array(z.string()),
  // Optional event_kind filter — defaults to all kinds.
  eventKinds: z.array(instanceEventKind),
  // Optional substring search against message. Uses lower(message) so the
  // ngrambf_v1 skip index on lower(message) can prune granules.
  search: z.string().nullable(),
  limit: z.int().default(50),
  // Composite cursor on (time, event_fingerprint). Time alone isn't unique
  // — krane batches multiple events into a single pod-watch tick and they
  // all land at the same millisecond, so a time-only cursor would
  // permanently drop the tail of any same-ms bucket that crosses a page
  // boundary. event_fingerprint breaks the tie deterministically and is
  // stable across retries (krane computes it from row content).
  // Both fields are set together: the first page passes both null, every
  // subsequent page echoes the last row's (time, event_fingerprint).
  cursorTime: z.int().nullable(),
  cursorFingerprint: z.string().nullable(),
});

export type InstanceEventsRequest = z.infer<typeof instanceEventsRequestSchema>;

export const instanceEvent = z.object({
  time: z.int(),
  workspace_id: z.string(),
  project_id: z.string(),
  app_id: z.string(),
  environment_id: z.string(),
  deployment_id: z.string(),
  pod_uid: z.string(),
  pod_name: z.string(),
  node_name: z.string(),
  container_name: z.string(),
  container_id: z.string(),
  restart_count: z.int(),
  // For 'waiting', the kubelet reason (CrashLoopBackOff, ImagePullBackOff, …)
  // lives in the `reason` column.
  event_kind: instanceEventKind,
  exit_code: z.int(),
  signal: z.int(),
  reason: z.string(),
  message: z.string(),
  region: z.string(),
  platform: z.string(),
  event_fingerprint: z.string(),
});

export type InstanceEvent = z.infer<typeof instanceEvent>;

export function getInstanceEvents(ch: Querier) {
  return async (args: InstanceEventsRequest) => {
    // Tenant-isolating predicates first so the primary key fully prunes
    // before the bloom filters narrow on deployment_id. Optional
    // filters use the same Nullable(...) pattern as runtime-logs so the
    // generated query plan stays consistent.
    const filterConditions = `
      workspace_id = {workspaceId: String}
      AND project_id = {projectId: String}
      AND (
        {environmentId: Nullable(String)} IS NULL
        OR environment_id = assumeNotNull({environmentId: Nullable(String)})
      )
      AND (
        {deploymentId: Nullable(String)} IS NULL
        OR deployment_id = assumeNotNull({deploymentId: Nullable(String)})
      )
      AND (
        {startTime: Nullable(UInt64)} IS NULL
        OR time >= assumeNotNull({startTime: Nullable(UInt64)})
      )
      AND (
        {endTime: Nullable(UInt64)} IS NULL
        OR time <= assumeNotNull({endTime: Nullable(UInt64)})
      )
      AND (
        CASE
          WHEN length({podUids: Array(String)}) > 0 THEN
            pod_uid IN {podUids: Array(String)}
          ELSE TRUE
        END
      )
      AND (
        CASE
          WHEN length({podNames: Array(String)}) > 0 THEN
            pod_name IN {podNames: Array(String)}
          ELSE TRUE
        END
      )
      AND (
        CASE
          WHEN length({regions: Array(String)}) > 0 THEN
            region IN {regions: Array(String)}
          ELSE TRUE
        END
      )
      AND (
        CASE
          WHEN length({eventKinds: Array(String)}) > 0 THEN
            event_kind IN {eventKinds: Array(String)}
          ELSE TRUE
        END
      )
      AND (
        {search: Nullable(String)} IS NULL
        OR {search: Nullable(String)} = ''
        OR positionCaseInsensitive(lower(message), lower(assumeNotNull({search: Nullable(String)}))) > 0
      )
    `;

    const eventsQuery = ch.query({
      query: `
        SELECT
          time, workspace_id, project_id, app_id, environment_id, deployment_id,
          pod_uid, pod_name, node_name, container_name, container_id, restart_count,
          event_kind, exit_code, signal, reason, message,
          region, platform, event_fingerprint
        FROM ${TABLE}
        WHERE ${filterConditions}
          AND (
            {cursorTime: Nullable(UInt64)} IS NULL
            OR time < assumeNotNull({cursorTime: Nullable(UInt64)})
            OR (
              time = assumeNotNull({cursorTime: Nullable(UInt64)})
              AND {cursorFingerprint: Nullable(String)} IS NOT NULL
              AND event_fingerprint < assumeNotNull({cursorFingerprint: Nullable(String)})
            )
          )
        ORDER BY time DESC, event_fingerprint DESC
        LIMIT {limit: Int}`,
      params: instanceEventsRequestSchema,
      schema: instanceEvent,
    });

    return { eventsQuery: eventsQuery(args) };
  };
}
