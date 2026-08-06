const SHA_PATTERN = /^[0-9a-f]{40}$/i;
const PR_URL = /^https?:\/\/github\.com\/[^/]+\/[^/]+\/pull\/\d+\/?$/;
const COMMIT_URL = /^https?:\/\/github\.com\/([^/]+\/[^/]+)\/commit\/([0-9a-f]{40})$/i;
const TREE_URL = /^https?:\/\/github\.com\/[^/]+\/[^/]+\/tree\/(.+)$/;

export type DeployGitSource = {
  branch?: string;
  commitSha?: string;
  repository?: string;
};

export class UnsupportedDeployRefError extends Error {}

const FORK_BRANCH_MESSAGE =
  "Deploying a branch from another repository is not supported. Paste the commit SHA instead.";

/**
 * Turns what a user typed into the git source deployments.createDeployment
 * accepts. The API resolves a branch to its head commit and fills the commit
 * metadata, so only the branch or SHA has to be identified here.
 *
 * Throws UnsupportedDeployRefError for refs the request cannot express: a pull
 * request URL, or another repository addressed by branch rather than by commit.
 */
export function parseDeployRef(raw: string): DeployGitSource {
  const trimmed = raw.trim();

  if (PR_URL.test(trimmed)) {
    throw new UnsupportedDeployRefError(
      "Pull request URLs are not supported. Paste the commit SHA you want to deploy.",
    );
  }

  const commit = trimmed.match(COMMIT_URL);
  if (commit) {
    return { commitSha: commit[2], repository: commit[1] };
  }

  const tree = trimmed.match(TREE_URL);
  if (tree) {
    throw new UnsupportedDeployRefError(FORK_BRANCH_MESSAGE);
  }

  // "owner:branch" addresses a branch on someone else's fork.
  const colonIdx = trimmed.indexOf(":");
  if (colonIdx > 0 && !trimmed.startsWith("http")) {
    const owner = trimmed.slice(0, colonIdx);
    const branch = trimmed.slice(colonIdx + 1);
    if (owner && branch && !owner.includes("/")) {
      throw new UnsupportedDeployRefError(FORK_BRANCH_MESSAGE);
    }
  }

  if (SHA_PATTERN.test(trimmed)) {
    return { commitSha: trimmed };
  }

  return { branch: trimmed };
}
