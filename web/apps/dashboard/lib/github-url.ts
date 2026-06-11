/**
 * Single source of truth for building github.com web links. Every builder returns
 * undefined when a required part is missing so callers can fall back to plain text
 * instead of rendering a broken link.
 */
import { buildUrl } from "@/lib/navigation/url";

const GITHUB_BASE = "https://github.com";

type RepoName = string | null | undefined;

export const githubUrl = {
  repo(repoFullName: RepoName): string | undefined {
    return repoFullName ? buildUrl({ base: GITHUB_BASE, segments: [repoFullName] }) : undefined;
  },

  commit(repoFullName: RepoName, sha: string | null | undefined): string | undefined {
    return repoFullName && sha
      ? buildUrl({ base: GITHUB_BASE, segments: [repoFullName, "commit", sha] })
      : undefined;
  },

  branch(repoFullName: RepoName, branch: string | null | undefined): string | undefined {
    return repoFullName && branch
      ? buildUrl({ base: GITHUB_BASE, segments: [repoFullName, "tree", branch] })
      : undefined;
  },

  pull(repoFullName: RepoName, prNumber: number | null | undefined): string | undefined {
    return repoFullName && prNumber != null
      ? buildUrl({ base: GITHUB_BASE, segments: [repoFullName, "pull", prNumber] })
      : undefined;
  },

  /**
   * Preferred link for a deployment's source: the PR when present, else the commit.
   * The PR lives on the base repo; a fork PR's commit lives on the fork, so commit
   * links resolve against `forkRepoFullName || repoFullName`.
   */
  deployment(args: {
    repoFullName: RepoName;
    forkRepoFullName: RepoName;
    prNumber: number | null | undefined;
    sha: string | null | undefined;
  }): string | undefined {
    const sourceRepo = args.forkRepoFullName || args.repoFullName;
    return (
      githubUrl.pull(args.repoFullName, args.prNumber) ?? githubUrl.commit(sourceRepo, args.sha)
    );
  },
};
