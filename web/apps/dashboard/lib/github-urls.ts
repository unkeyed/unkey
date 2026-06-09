/**
 * Single source of truth for building github.com web links. Every builder
 * returns undefined when a required part is missing so callers can fall back to
 * plain text instead of rendering a broken link.
 *
 * Commit and branch links should use the fork-aware source repo
 * (`forkRepositoryFullName || repositoryFullName`); PR links always use the base
 * repo, since the PR lives on the repo it targets.
 */
const GITHUB_BASE = "https://github.com";

type RepoName = string | null | undefined;

export function githubRepoUrl(repoFullName: RepoName): string | undefined {
  return repoFullName ? `${GITHUB_BASE}/${repoFullName}` : undefined;
}

export function githubCommitUrl(
  repoFullName: RepoName,
  sha: string | null | undefined,
): string | undefined {
  return repoFullName && sha ? `${GITHUB_BASE}/${repoFullName}/commit/${sha}` : undefined;
}

export function githubBranchUrl(
  repoFullName: RepoName,
  branch: string | null | undefined,
): string | undefined {
  return repoFullName && branch ? `${GITHUB_BASE}/${repoFullName}/tree/${branch}` : undefined;
}

export function githubPullUrl(
  repoFullName: RepoName,
  prNumber: number | null | undefined,
): string | undefined {
  return repoFullName && prNumber != null
    ? `${GITHUB_BASE}/${repoFullName}/pull/${prNumber}`
    : undefined;
}

/**
 * Preferred link for a deployment's source: the PR when present, else the commit.
 * The PR lives on the base repo; a fork PR's commit lives on the fork, so commit
 * links resolve against `forkRepoFullName || repoFullName`.
 */
export function githubDeploymentUrl(args: {
  repoFullName: RepoName;
  forkRepoFullName: RepoName;
  prNumber: number | null | undefined;
  sha: string | null | undefined;
}): string | undefined {
  const sourceRepo = args.forkRepoFullName || args.repoFullName;
  return githubPullUrl(args.repoFullName, args.prNumber) ?? githubCommitUrl(sourceRepo, args.sha);
}
