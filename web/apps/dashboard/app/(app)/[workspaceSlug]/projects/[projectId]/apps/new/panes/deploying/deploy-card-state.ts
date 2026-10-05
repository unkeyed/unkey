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

export const watchStatus: Record<RunOutcome, string> = {
  running: "Your app is building and deploying.",
  live: "Your app is live.",
  failed: "Deployment failed. Open the failed step to see why.",
  blocked: "This deployment needs approval. View the deployment to approve it.",
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
      title: stage.key === "building" ? "Build logs" : stage.label,
      meta: stage.durationMs === null ? (stage.note ?? "") : formatStageDuration(stage.durationMs),
      expandable: detail.type !== "none",
      fillsCard: detail.type === "logs",
      detail,
    };
  });
}
