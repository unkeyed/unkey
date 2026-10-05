import type { InstanceSummary } from "./instance-state";
import {
  type RunOutcome,
  type RunView,
  type Stage,
  type StageKey,
  formatStageDuration,
} from "./run-model";

export type StageDetail =
  | { type: "none" }
  | { type: "logs"; error: string | null }
  | { type: "crash" }
  | { type: "error"; message: string };

export type StageRow = {
  key: StageKey;
  stage: Stage;
  title: string;
  meta: string;
  expandable: boolean;
  fillsCard: boolean;
  detail: StageDetail;
};

type WatchFooter = "none" | "continue" | "failed" | "blocked";

export const watchStatus: Record<RunOutcome, string> = {
  running: "Your app is building and deploying.",
  live: "Your app is live.",
  failed: "Deployment failed. Open the failed step to see why.",
  blocked: "This deployment needs approval. View the deployment to approve it.",
};

export const watchFooter: Record<RunOutcome, WatchFooter> = {
  running: "none",
  live: "continue",
  failed: "failed",
  blocked: "blocked",
};

const stageTitle: Record<StageKey, string | null> = {
  queued: null,
  building: "Build logs",
  deploying: null,
  network: null,
  finalizing: null,
};

function stageDetail(stage: Stage, view: RunView, crashed: boolean): StageDetail {
  if (stage.key === "building") {
    return {
      type: "logs",
      error:
        stage.state === "failed"
          ? (view.firstError ??
            stage.error ??
            "The build failed without an error message. Check the build logs, then redeploy.")
          : null,
    };
  }
  if (stage.state !== "failed") {
    return { type: "none" };
  }
  if (stage.key === "deploying" && crashed) {
    return { type: "crash" };
  }
  return {
    type: "error",
    message:
      stage.error ??
      view.firstError ??
      "The deployment failed without an error message. View the deployment, then redeploy.",
  };
}

export function isInstanceCrash(view: RunView, instances: InstanceSummary | null): boolean {
  return view.failedStage?.key === "deploying" && instances?.state === "unhealthy";
}

export function stageRows(view: RunView, crashed: boolean): StageRow[] {
  return view.stages.map((stage) => {
    const detail = stageDetail(stage, view, crashed);
    return {
      key: stage.key,
      stage,
      title: stageTitle[stage.key] ?? stage.label,
      meta: stage.durationMs === null ? (stage.note ?? "") : formatStageDuration(stage.durationMs),
      expandable: detail.type !== "none",
      fillsCard: detail.type === "logs",
      detail,
    };
  });
}

export type ResultValue =
  | { type: "domain" }
  | { type: "pending"; text: string }
  | { type: "mono"; text: string; tone: "plain" | "error" };

type ResultRowState = { label: string; value: ResultValue };

type ResultInput = {
  hasDomain: boolean;
  instances: { text: string; tone: "plain" | "error" | "warn" } | null;
  gitBranch: string | null;
  gitCommitSha: string | null;
};

export function resultRows({
  hasDomain,
  instances,
  gitBranch,
  gitCommitSha,
}: ResultInput): ResultRowState[] {
  const rows: ResultRowState[] = [
    {
      label: "Domain",
      value: hasDomain ? { type: "domain" } : { type: "pending", text: "Assigning domain…" },
    },
  ];
  if (instances) {
    rows.push({
      label: "Instances",
      value: {
        type: "mono",
        text: instances.text,
        tone: instances.tone === "error" ? "error" : "plain",
      },
    });
  }
  if (gitCommitSha) {
    rows.push({
      label: "Commit",
      value: {
        type: "mono",
        text: `${gitBranch ?? ""} · ${gitCommitSha.slice(0, 7)}`,
        tone: "plain",
      },
    });
  }
  return rows;
}
