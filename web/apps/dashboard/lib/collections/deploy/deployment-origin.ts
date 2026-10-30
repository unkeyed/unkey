import type { DeploymentSummary } from "./deployments";

type Trigger = DeploymentSummary["trigger"];

export type GitTrigger = Extract<Trigger, "github" | "unknown">;

export function isGitTrigger(trigger: Trigger): trigger is GitTrigger {
  return trigger === "github" || trigger === "unknown";
}

export type GitAuthor = { handle: string | null; avatarUrl: string | null };

export function gitAuthorOf(
  deployment: Pick<
    DeploymentSummary,
    "source" | "trigger" | "gitCommitAuthorHandle" | "gitCommitAuthorAvatarUrl"
  >,
): GitAuthor | null {
  if (deployment.source !== "git" || !isGitTrigger(deployment.trigger)) {
    return null;
  }
  return {
    handle: deployment.gitCommitAuthorHandle,
    avatarUrl: deployment.gitCommitAuthorAvatarUrl,
  };
}
