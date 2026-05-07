import type { getPullRequest } from "@/lib/github";

export type DeployRef =
  | { kind: "pr"; prNumber: number; sourceRepo: string }
  | { kind: "sha"; sha: string; sourceRepo?: string }
  | { kind: "branch"; branch: string; sourceRepo?: string };

const SHA_PATTERN = /^[0-9a-f]{40}$/i;

export function parseDeployRef(raw: string): DeployRef {
  const trimmed = raw.trim();

  const prMatch = trimmed.match(/^https?:\/\/github\.com\/([^/]+\/[^/]+)\/pull\/(\d+)\/?$/);
  if (prMatch) {
    return { kind: "pr", prNumber: Number.parseInt(prMatch[2], 10), sourceRepo: prMatch[1] };
  }

  const treeMatch = trimmed.match(/^https?:\/\/github\.com\/([^/]+\/[^/]+)\/tree\/(.+)$/);
  if (treeMatch) {
    return { kind: "branch", branch: treeMatch[2], sourceRepo: treeMatch[1] };
  }

  const commitMatch = trimmed.match(
    /^https?:\/\/github\.com\/([^/]+\/[^/]+)\/commit\/([0-9a-f]{40})$/i,
  );
  if (commitMatch) {
    return { kind: "sha", sha: commitMatch[2], sourceRepo: commitMatch[1] };
  }

  const colonIdx = trimmed.indexOf(":");
  if (colonIdx > 0 && !trimmed.startsWith("http")) {
    const owner = trimmed.slice(0, colonIdx);
    const branch = trimmed.slice(colonIdx + 1);
    if (owner && branch && !owner.includes("/")) {
      return { kind: "branch", branch, sourceRepo: owner };
    }
  }

  if (SHA_PATTERN.test(trimmed)) {
    return { kind: "sha", sha: trimmed };
  }

  return { kind: "branch", branch: trimmed };
}

export function detectForkRepo(pr: Awaited<ReturnType<typeof getPullRequest>>): string | undefined {
  if (pr.head.repo?.fork === true && pr.head.repo.full_name !== pr.base.repo.full_name) {
    return pr.head.repo.full_name;
  }
  return undefined;
}

export function resolveSourceRepo(
  sourceRepo: string,
  baseRepoFullName: string,
): string | undefined {
  const baseRepoName = baseRepoFullName.split("/")[1];
  if (sourceRepo.includes("/")) {
    const sourceRepoName = sourceRepo.split("/")[1];
    if (sourceRepoName !== baseRepoName) {
      return undefined;
    }
    return sourceRepo !== baseRepoFullName ? sourceRepo : undefined;
  }
  return `${sourceRepo}/${baseRepoName}`;
}
