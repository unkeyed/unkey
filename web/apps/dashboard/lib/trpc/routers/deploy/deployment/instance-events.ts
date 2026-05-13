import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { instanceEventKind } from "@unkey/clickhouse";
import { z } from "zod";

// getDeploymentInstanceEvents lists container lifecycle events (running,
// terminations, and waiting transitions) scoped to a project — narrowable
// to a specific deployment.
//
// Project-wide is the default for the runtime-logs page (no deploymentId
// filter set yet); the deployment-detail surfaces pass the deploymentId so
// the same query powers both. The ClickHouse table is keyed by
// (workspace_id, project_id, app_id, environment_id, time, deployment_id)
// so the project-only path still range-prunes by partition + primary key.
export const getDeploymentInstanceEvents = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      projectId: z.string(),
      // Optional narrowing — pass when scoping to a single deployment.
      deploymentId: z.string().nullable().default(null),
      environmentId: z.string().nullable().default(null),
      // Optional pod_uid scope used by the logs viewer enrichment when the
      // user clicks "View logs" on a specific event row.
      podUids: z.array(z.string()).default([]),
      // Optional pod name filter — the runtime-logs filter UI exposes
      // "instance ID" which corresponds to k8s pod_name in the events
      // table. Empty array = all pods.
      podNames: z.array(z.string()).default([]),
      // Optional region filter — mirrors the runtime-logs region filter
      // so events follow the same scope as the visible logs.
      regions: z.array(z.string()).default([]),
      // Optional kind filter. For "waiting", consumers can additionally
      // filter on the kubelet reason (CrashLoopBackOff, ImagePullBackOff,
      // …) via the message search since reason is part of the row.
      eventKinds: z.array(instanceEventKind).default([]),
      // Optional substring search across the event message.
      search: z.string().nullable().default(null),
      // Optional time bounds (unix milliseconds). Used by the logs viewer
      // to scope to the same window the user is reading.
      startTime: z.number().int().nullable().default(null),
      endTime: z.number().int().nullable().default(null),
      limit: z.number().int().min(1).max(200).default(50),
      // Composite cursor on (time, event_fingerprint). Both are echoed back
      // from a previous page's nextCursor. Both null = first page. The
      // fingerprint tiebreaker keeps same-millisecond batches from being
      // dropped between pages — krane batches one pod-watch tick into a
      // single RPC and every event in that batch lands at the same ms.
      cursorTime: z.number().int().nullable().default(null),
      cursorFingerprint: z.string().nullable().default(null),
    }),
  )
  .output(
    z.object({
      events: z.array(
        z.object({
          time: z.number(),
          podUid: z.string(),
          podName: z.string(),
          nodeName: z.string(),
          containerName: z.string(),
          containerId: z.string(),
          restartCount: z.number(),
          eventKind: instanceEventKind,
          exitCode: z.number(),
          signal: z.number(),
          reason: z.string(),
          message: z.string(),
          region: z.string(),
          eventFingerprint: z.string(),
        }),
      ),
      // Cursor for the next page. Null means no more results.
      nextCursor: z
        .object({
          time: z.number().int(),
          fingerprint: z.string(),
        })
        .nullable(),
    }),
  )
  .query(async ({ ctx, input }) => {
    // Verify the project belongs to the caller's workspace. Without this
    // check, an authenticated user could query events for any project just
    // by guessing the projectId. The deployment-scoped variant gets this
    // check transitively when we look up the deployment row, but the
    // project-only path needs it explicit.
    const project = await db.query.projects.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      columns: { id: true, workspaceId: true },
    });
    if (!project) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Project not found" });
    }

    const { eventsQuery } = await clickhouse.instanceEvents.list({
      workspaceId: project.workspaceId,
      projectId: project.id,
      environmentId: input.environmentId,
      deploymentId: input.deploymentId,
      podUids: input.podUids,
      podNames: input.podNames,
      regions: input.regions,
      eventKinds: input.eventKinds,
      search: input.search,
      startTime: input.startTime,
      endTime: input.endTime,
      limit: input.limit,
      cursorTime: input.cursorTime,
      cursorFingerprint: input.cursorFingerprint,
    });

    const result = await eventsQuery;
    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch instance events",
      });
    }

    const rows = result.val;
    const events = rows.map((r) => ({
      time: r.time,
      podUid: r.pod_uid,
      podName: r.pod_name,
      nodeName: r.node_name,
      containerName: r.container_name,
      containerId: r.container_id,
      restartCount: r.restart_count,
      eventKind: r.event_kind,
      exitCode: r.exit_code,
      signal: r.signal,
      reason: r.reason,
      message: r.message,
      region: r.region,
      eventFingerprint: r.event_fingerprint,
    }));

    // The cursor echoes the last row's (time, event_fingerprint) when we
    // filled the page; if we got fewer rows than requested we know there's
    // no next page. Echoing the fingerprint along with time is what lets
    // the next call's WHERE clause skip exactly the rows we already saw,
    // including any same-millisecond siblings that would otherwise be lost.
    const nextCursor =
      events.length === input.limit
        ? {
            time: events[events.length - 1].time,
            fingerprint: events[events.length - 1].eventFingerprint,
          }
        : null;

    return { events, nextCursor };
  });
