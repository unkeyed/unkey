import type { getPullRequest } from "@/lib/github";
import { TRPCError } from "@trpc/server";

export type RepoConn = { installationId: number; repositoryFullName: string };

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

// A GitHub owner/repo full name: only the characters GitHub actually allows in
// account and repository names. The build worker interpolates this fork name
// straight into a git context URL ("https://github.com/<fork>.git#<ref>"), so a
// name containing "#", "?", or whitespace could smuggle a fragment or query into
// that URL and alter which ref BuildKit checks out. Constraining the charset
// here keeps a user-typed source repo from reaching the worker as a malformed URL.
const REPO_FULL_NAME = /^[A-Za-z0-9._-]+\/[A-Za-z0-9._-]+$/;

export function resolveSourceRepo(
  sourceRepo: string,
  baseRepoFullName: string,
): string | undefined {
  // GitHub owner/repo names are case-insensitive, so compare case-folded while
  // preserving the caller's casing in the returned value.
  const baseRepoName = baseRepoFullName.split("/")[1];
  let candidate: string;
  if (sourceRepo.includes("/")) {
    const sourceRepoName = sourceRepo.split("/")[1];
    if (sourceRepoName.toLowerCase() !== baseRepoName.toLowerCase()) {
      return undefined;
    }
    candidate = sourceRepo;
  } else {
    candidate = `${sourceRepo}/${baseRepoName}`;
  }
  if (
    !REPO_FULL_NAME.test(candidate) ||
    candidate.toLowerCase() === baseRepoFullName.toLowerCase()
  ) {
    return undefined;
  }
  return candidate;
}

export function validateSourceRepo(sourceRepo: string, repoConn: RepoConn): string {
  const fork = resolveSourceRepo(sourceRepo, repoConn.repositoryFullName);
  if (fork) {
    return fork;
  }

  // resolveSourceRepo returns undefined both when the ref points at the base
  // repo itself (a legitimate non-fork deploy) and when it is malformed or not
  // a fork of this repo. Only the former may proceed silently: rejecting the
  // latter stops a bad owner-only ref (e.g. "evil owner") from being dropped
  // and reinterpreted as a fully trusted build of the base branch.
  const baseName = repoConn.repositoryFullName.split("/")[1];
  const candidate = sourceRepo.includes("/") ? sourceRepo : `${sourceRepo}/${baseName}`;
  if (candidate.toLowerCase() === repoConn.repositoryFullName.toLowerCase()) {
    return "";
  }

  throw new TRPCError({
    code: "BAD_REQUEST",
    message: `Repository "${sourceRepo}" is not a fork of "${repoConn.repositoryFullName}"`,
  });
}
