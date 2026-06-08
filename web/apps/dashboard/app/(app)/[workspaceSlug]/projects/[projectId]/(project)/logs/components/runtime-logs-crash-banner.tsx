"use client";

import { trpc } from "@/lib/trpc/client";
import { TriangleWarning2 } from "@unkey/icons";
import { TimestampInfo } from "@unkey/ui";
import { useParams } from "next/navigation";
import { useMemo } from "react";
import { useRuntimeLogsFilters } from "../hooks/use-runtime-logs-filters";

// RuntimeLogsCrashBanner is the summary header shown above the logs
// table when the user has scoped the view to a specific deployment AND
// that deployment has crashed in the visible time window. Surfaces the
// headline diagnostic (count, last reason, when, deployment id) at the
// top so a broken deploy is unmissable.
//
// Stays hidden on the project-wide view: a preview branch crashing
// somewhere across the project is not actionable from here, and would
// just be noise for users debugging a different deployment.
export function RuntimeLogsCrashBanner() {
  const { filters } = useRuntimeLogsFilters();
  const params = useParams<{ projectId: string }>();
  const deploymentIdFilter = filters.find((f) => f.field === "deploymentId");
  const deploymentId = deploymentIdFilter ? String(deploymentIdFilter.value) : null;

  // Mirror the runtime-logs filter shape so the events count narrows in
  // lockstep with the visible logs. The user's mental model: "what
  // crashed in the slice I'm currently viewing." Region + instanceId
  // filters can be repeated, so we collect all matching values into
  // arrays. Severity isn't forwarded — events don't carry log severity,
  // and a user filtering for ERROR logs probably still wants to see
  // crashes summarised even if their stdout was at INFO.
  const queryInput = useMemo(() => {
    const environmentId = filters.find((f) => f.field === "environmentId");
    const messageFilter = filters.find((f) => f.field === "message");
    const startFilter = filters.find((f) => f.field === "startTime");
    const endFilter = filters.find((f) => f.field === "endTime");
    const regions = filters.filter((f) => f.field === "region").map((f) => String(f.value));
    const podNames = filters.filter((f) => f.field === "instanceId").map((f) => String(f.value));
    return {
      projectId: params.projectId,
      deploymentId,
      environmentId: environmentId ? String(environmentId.value) : null,
      podUids: [],
      podNames,
      regions,
      eventKinds: [],
      search: messageFilter ? String(messageFilter.value) : null,
      startTime: startFilter ? Number(startFilter.value) : Date.now() - 6 * 60 * 60 * 1000,
      endTime: endFilter ? Number(endFilter.value) : Date.now(),
      limit: 200,
      cursorTime: null,
    };
  }, [filters, params.projectId, deploymentId]);

  const { data } = trpc.deploy.deployment.instanceEvents.useQuery(queryInput, {
    // Banner is deployment-scoped only — no fetch on the project-wide
    // view, so we don't accidentally alarm users about preview branches
    // they aren't looking at.
    enabled: Boolean(params.projectId) && deploymentId !== null,
    refetchInterval: 10_000,
    refetchOnWindowFocus: false,
  });

  const summary = useMemo(() => summarize(data?.events ?? []), [data]);

  // Hide on the project-wide view (no deployment in filter), or when the
  // scoped deployment hasn't crashed in the visible window.
  if (!deploymentId || !summary) {
    return null;
  }

  return (
    // Sticky-positioned so the headline stays visible as the user scrolls
    // through the log table. `top-0` anchors to the parent's scroll
    // container; the high z-index sits above the virtualized rows so
    // nothing renders over it.
    <div className="sticky top-0 z-10 mx-2 my-2 flex items-center gap-3 rounded-lg border border-errorA-5 bg-errorA-2 px-3 py-2 text-[12px] text-error-11 backdrop-blur-sm">
      <TriangleWarning2 iconSize="md-regular" className="text-error-9 shrink-0" />
      <div className="flex flex-col gap-0.5 min-w-0 flex-1">
        <div className="font-medium">
          {summary.terminations === 1 ? "1 crash" : `${summary.terminations} crashes`} in{" "}
          <span className="font-mono">{deploymentId}</span>
          {summary.distinctPods > 1 && (
            <span className="text-grayA-10 font-normal"> · across {summary.distinctPods} pods</span>
          )}
        </div>
        <div className="text-grayA-11 truncate font-mono text-[11px]">
          last:{" "}
          <span className="text-error-11">
            {summary.lastReason}
            {summary.lastExitCode !== null && summary.lastExitCode !== 0 && (
              <> · exit={summary.lastExitCode}</>
            )}
          </span>{" "}
          · <TimestampInfo value={summary.lastTime} displayType="relative" />
        </div>
      </div>
    </div>
  );
}

type RawEvent = {
  time: number;
  eventKind: string;
  reason: string;
  exitCode: number;
  podUid: string;
};

type CrashSummary = {
  terminations: number;
  distinctPods: number;
  lastTime: number;
  lastReason: string;
  lastExitCode: number | null;
};

// summarize folds the events list into the headline numbers the banner
// shows. Returns null when there are no terminated events in the
// window, which hides the banner entirely. We count terminations only —
// the throttle/backoff state is kubelet's observation of the same
// terminations and would double-count.
//
// Pod names are intentionally absent from the summary; they're an
// internal k8s implementation detail. The user-facing instance ID
// (ins_*) lives on the runtime-logs row + per-instance details panel
// for users who need to drill into a specific replica.
function summarize(events: RawEvent[]): CrashSummary | null {
  const terms = events.filter((e) => e.eventKind === "terminated");
  if (terms.length === 0) {
    return null;
  }
  const last = terms[0]; // events arrive DESC by time
  const distinctPods = new Set(terms.map((e) => e.podUid)).size;
  return {
    terminations: terms.length,
    distinctPods,
    lastTime: last.time,
    lastReason: last.reason || "Error",
    lastExitCode: last.exitCode,
  };
}
