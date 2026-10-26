import type { DeployMarker, DeploySource, Grid } from "@/lib/metrics/app-metrics";
import type { MarkerDeployment, ProductionRouting } from "./scope";

export function toDeployMarkers(
  rows: ReadonlyArray<MarkerDeployment>,
  grid: Pick<Grid, "startMs" | "endMs">,
  routing: Pick<ProductionRouting, "isRolledBack" | "currentReadyAtMs"> | null,
): DeployMarker[] {
  const ready = rows
    .flatMap((row) =>
      row.firstReadyAt !== null && row.firstReadyAt < grid.endMs
        ? [{ row, readyAtMs: row.firstReadyAt }]
        : [],
    )
    .sort((a, b) => a.readyAtMs - b.readyAtMs || a.row.id.localeCompare(b.row.id));

  const firstInRange = ready.findIndex((d) => d.readyAtMs >= grid.startMs);
  const from = firstInRange === -1 ? ready.length - 1 : firstInRange - 1;

  const newestReadyAtMs = ready.at(-1)?.readyAtMs;
  // Spans assume each deploy served until the next one became ready. A rollback
  // or a promote of an older deploy breaks that, and no routing history is
  // stored, so spans are only drawn while the newest ready deploy is current.
  const servingKnown =
    routing !== null &&
    !routing.isRolledBack &&
    routing.currentReadyAtMs !== null &&
    (newestReadyAtMs === undefined || routing.currentReadyAtMs >= newestReadyAtMs);

  return ready.slice(Math.max(from, 0)).map((d, i, kept) => ({
    deploymentId: d.row.id,
    readyAtMs: d.readyAtMs,
    serving: servingKnown ? { untilMs: kept[i + 1]?.readyAtMs ?? null } : null,
    source: deploySourceOf(d.row),
  }));
}

type SourceColumns = Pick<
  MarkerDeployment,
  | "gitCommitSha"
  | "gitCommitMessage"
  | "gitCommitAuthorHandle"
  | "gitCommitAuthorAvatarUrl"
  | "imageRequested"
>;

export function deploySourceOf(row: SourceColumns): DeploySource | null {
  if (row.gitCommitSha) {
    return {
      kind: "git",
      sha: row.gitCommitSha,
      message: row.gitCommitMessage,
      authorHandle: row.gitCommitAuthorHandle,
      avatarUrl: row.gitCommitAuthorAvatarUrl,
    };
  }
  if (row.imageRequested) {
    return { kind: "image", ref: row.imageRequested };
  }
  return null;
}
