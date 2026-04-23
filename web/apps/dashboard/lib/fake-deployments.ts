import type { Deployment } from "@/lib/collections/deploy/deployments";

/**
 * Deterministic mock deployments shown in v2b when a project has zero
 * real deployments, so Dave can see the 3-panel layout populated while
 * apps aren't a real backend concept yet.
 *
 * The IDs deliberately start with `dep_fake_` so any downstream code
 * (the detail page, navigation) can trivially tell a mock from a real
 * deployment and render a placeholder for the former.
 */
const MOCK_COMMITS = [
  "fix: handle stale sessions from upstream",
  "feat: wire up audit log export",
  "chore: bump @unkey/ui",
  "revert: roll back idempotency change",
] as const;

const MOCK_BRANCHES = ["main", "main", "feat/audit-export", "main"] as const;
const MOCK_STATUSES: Deployment["status"][] = ["ready", "building", "failed", "ready"];
const MOCK_AUTHORS: Array<{ handle: string; avatar: string }> = [
  { handle: "davehawkins", avatar: "https://github.com/davehawkins.png" },
  { handle: "chronark", avatar: "https://github.com/chronark.png" },
  { handle: "james", avatar: "https://github.com/jamesperkins.png" },
  { handle: "ozperzzz", avatar: "https://github.com/ozperzzz.png" },
];

export const FAKE_DEPLOYMENT_PREFIX = "dep_fake_";

export function isFakeDeployment(deployment: { id: string }): boolean {
  return deployment.id.startsWith(FAKE_DEPLOYMENT_PREFIX);
}

function hexSha(seed: string, idx: number): string {
  // Cheap deterministic hex-ish string derived from a seed + index.
  let h = 0;
  for (const char of `${seed}:${idx}`) {
    h = (h * 31 + char.charCodeAt(0)) | 0;
  }
  const hex = Math.abs(h).toString(16).padStart(7, "0").slice(0, 7);
  return hex;
}

export function generateFakeDeployments(
  projectId: string,
  environmentId = "env_fake_prod",
): Deployment[] {
  const now = Date.now();
  const hourMs = 60 * 60 * 1000;
  return MOCK_COMMITS.map((message, i) => {
    const author = MOCK_AUTHORS[i % MOCK_AUTHORS.length];
    const status = MOCK_STATUSES[i];
    const createdAt = now - (i + 1) * hourMs * (i === 2 ? 24 : 1);
    return {
      id: `${FAKE_DEPLOYMENT_PREFIX}${projectId}_${i}`,
      projectId,
      environmentId,
      gitCommitSha: hexSha(projectId, i),
      gitBranch: MOCK_BRANCHES[i],
      gitCommitMessage: message,
      gitCommitAuthorHandle: author.handle,
      gitCommitAuthorAvatarUrl: author.avatar,
      gitCommitTimestamp: createdAt,
      prNumber: i === 2 ? 142 : null,
      forkRepositoryFullName: null,
      hasOpenApiSpec: i % 2 === 0,
      status,
      instances:
        status === "ready"
          ? [
              {
                id: `inst_fake_${projectId}_${i}_a`,
                region: { id: "de-fra", name: "Frankfurt", platform: "aws" },
                flagCode: "de" as const,
              },
              {
                id: `inst_fake_${projectId}_${i}_b`,
                region: { id: "us-east", name: "Virginia", platform: "aws" },
                flagCode: "us" as const,
              },
            ]
          : [],
      cpuMillicores: 500,
      memoryMib: 512,
      storageMib: 1024,
      createdAt,
    } satisfies Deployment;
  });
}
