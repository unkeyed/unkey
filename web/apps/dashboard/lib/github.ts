import crypto from "node:crypto";
import { githubAppEnv } from "@/lib/env";
import { z } from "zod";

const gitHubRepositorySchema = z.object({
  id: z.number(),
  name: z.string(),
  full_name: z.string(),
  private: z.boolean(),
  html_url: z.string(),
  default_branch: z.string(),
  pushed_at: z.string().nullable(),
  language: z.string().nullable(),
});

export type GitHubRepository = z.infer<typeof gitHubRepositorySchema>;

const installationAccessTokenSchema = z.object({
  token: z.string(),
  expires_at: z.string(),
});

const installationRepositoriesSchema = z.object({
  repositories: z.array(gitHubRepositorySchema),
});

const repositoryTreeSchema = z.object({
  tree: z.array(
    z.object({
      path: z.string(),
      type: z.string(),
    }),
  ),
  truncated: z.boolean().optional(),
});

const repositoryBranchesSchema = z.array(
  z.object({
    name: z.string(),
  }),
);

const GITHUB_API_HEADERS = {
  Accept: "application/vnd.github+json",
  "X-GitHub-Api-Version": "2022-11-28",
} as const;

async function fetchGitHubApi(url: string, token: string): Promise<unknown> {
  const response = await fetch(url, {
    headers: {
      Authorization: `Bearer ${token}`,
      ...GITHUB_API_HEADERS,
    },
  });

  if (!response.ok) {
    const body = await response.text();
    throw new Error(`GitHub API error ${response.status}: ${body}`);
  }

  return response.json();
}

function base64UrlEncode(data: string): string {
  return Buffer.from(data).toString("base64url");
}

function generateAppJWT(): string {
  const env = githubAppEnv();
  if (!env) {
    throw new Error("GitHub App environment not configured");
  }

  const header = { alg: "RS256", typ: "JWT" };
  const now = Math.floor(Date.now() / 1000);
  const payload = {
    iat: now - 60, // 60 seconds in the past for clock drift
    exp: now + 600, // 10 minutes max
    iss: env.GITHUB_APP_ID,
  };

  const encodedHeader = base64UrlEncode(JSON.stringify(header));
  const encodedPayload = base64UrlEncode(JSON.stringify(payload));
  const signatureInput = `${encodedHeader}.${encodedPayload}`;

  const sign = crypto.createSign("RSA-SHA256");
  sign.update(signatureInput);
  const signature = sign.sign(env.UNKEY_GITHUB_PRIVATE_KEY_PEM, "base64url");

  return `${signatureInput}.${signature}`;
}

export async function getInstallationAccessToken(
  installationId: number,
): Promise<{ token: string; expires_at: string }> {
  const jwt = generateAppJWT();

  const response = await fetch(
    `https://api.github.com/app/installations/${installationId}/access_tokens`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${jwt}`,
        ...GITHUB_API_HEADERS,
      },
    },
  );

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to get installation access token: ${error}`);
  }

  return installationAccessTokenSchema.parse(await response.json());
}

export async function getInstallationRepositories(
  installationId: number,
): Promise<GitHubRepository[]> {
  const { token } = await getInstallationAccessToken(installationId);

  const allRepositories: GitHubRepository[] = [];
  let page = 1;
  const perPage = 100;

  while (true) {
    const data = installationRepositoriesSchema.parse(
      await fetchGitHubApi(
        `https://api.github.com/installation/repositories?per_page=${perPage}&page=${page}`,
        token,
      ),
    );
    allRepositories.push(...data.repositories);

    if (data.repositories.length < perPage) {
      break;
    }
    page++;
  }

  return allRepositories;
}

export async function getRepositoryTree(
  installationId: number,
  owner: string,
  repo: string,
  branch: string,
): Promise<{ tree: Array<{ path: string; type: string }>; truncated: boolean }> {
  const { token } = await getInstallationAccessToken(installationId);

  const data = repositoryTreeSchema.parse(
    await fetchGitHubApi(
      `https://api.github.com/repos/${owner}/${repo}/git/trees/${branch}?recursive=1`,
      token,
    ),
  );

  return { tree: data.tree, truncated: data.truncated ?? false };
}

/**
 * Check whether a specific file exists in a repository using the Contents API.
 * Returns true if the file exists, false otherwise.
 */
export async function checkFileExists(
  installationId: number,
  owner: string,
  repo: string,
  branch: string,
  path: string,
): Promise<boolean> {
  const { token } = await getInstallationAccessToken(installationId);

  const response = await fetch(
    `https://api.github.com/repos/${owner}/${repo}/contents/${path}?ref=${branch}`,
    {
      method: "HEAD",
      headers: {
        Authorization: `Bearer ${token}`,
        ...GITHUB_API_HEADERS,
      },
    },
  );

  return response.ok;
}

const repositoryEventSchema = z.object({
  type: z.string(),
  created_at: z.string(),
  payload: z
    .object({
      ref: z.string().optional(),
      size: z.number().optional(),
    })
    // GitHub event payloads vary by type; passthrough avoids parse failures on extra fields
    .passthrough(),
});

const repositoryEventsSchema = z.array(repositoryEventSchema);

export const MAX_BRANCHES = 20;

export type BranchActivity = {
  name: string;
  pushCount: number;
  commitCount: number;
  lastPushDate: string;
};

export async function getMostActiveBranches(
  installationId: number,
  owner: string,
  repo: string,
): Promise<BranchActivity[]> {
  const { token } = await getInstallationAccessToken(installationId);

  const allEvents: z.infer<typeof repositoryEventSchema>[] = [];
  for (let page = 1; page <= 3; page++) {
    try {
      const raw = await fetchGitHubApi(
        `https://api.github.com/repos/${owner}/${repo}/events?per_page=100&page=${page}`,
        token,
      );
      // safeParse so malformed responses don't blow up the whole flow
      const parsed = repositoryEventsSchema.safeParse(raw);
      if (!parsed.success) {
        break;
      }
      allEvents.push(...parsed.data);
      // Fewer results than the page size means we've reached the last page
      if (parsed.data.length < 100) {
        break;
      }
    } catch {
      // GitHub API errors (rate limit, 404, etc.) — return whatever we collected so far
      break;
    }
  }

  const pushEvents = allEvents.filter((e) => e.type === "PushEvent" && e.payload.ref);

  const branchMap = new Map<
    string,
    { pushCount: number; commitCount: number; lastPushDate: string }
  >();

  for (const event of pushEvents) {
    const ref = event.payload.ref;
    if (!ref) {
      continue;
    }
    const branchName = ref.replace("refs/heads/", "");
    const existing = branchMap.get(branchName);
    const commits = event.payload.size ?? 0;

    if (existing) {
      existing.pushCount += 1;
      existing.commitCount += commits;
      if (event.created_at > existing.lastPushDate) {
        existing.lastPushDate = event.created_at;
      }
    } else {
      branchMap.set(branchName, {
        pushCount: 1,
        commitCount: commits,
        lastPushDate: event.created_at,
      });
    }
  }

  const branches: BranchActivity[] = [];
  for (const [name, data] of branchMap) {
    branches.push({ name, ...data });
  }

  branches.sort((a, b) => {
    if (a.commitCount !== b.commitCount) {
      return b.commitCount - a.commitCount;
    }
    return b.lastPushDate.localeCompare(a.lastPushDate);
  });

  return branches.slice(0, MAX_BRANCHES);
}

export async function getRepositoryBranches(
  installationId: number,
  owner: string,
  repo: string,
  perPage = 100,
): Promise<Array<{ name: string }>> {
  const { token } = await getInstallationAccessToken(installationId);

  return repositoryBranchesSchema.parse(
    await fetchGitHubApi(
      `https://api.github.com/repos/${owner}/${repo}/branches?per_page=${perPage}`,
      token,
    ),
  );
}

const matchingRefsSchema = z.array(
  z.object({
    ref: z.string(),
  }),
);

export async function searchBranchesByPrefix(
  installationId: number,
  owner: string,
  repo: string,
  prefix: string,
  limit = 30,
): Promise<Array<{ name: string }>> {
  const { token } = await getInstallationAccessToken(installationId);
  const data = matchingRefsSchema.parse(
    await fetchGitHubApi(
      `https://api.github.com/repos/${owner}/${repo}/git/matching-refs/heads/${prefix.split("/").map(encodeURIComponent).join("/")}`,
      token,
    ),
  );
  return data.slice(0, limit).map((ref) => ({ name: ref.ref.replace("refs/heads/", "") }));
}

export async function getRepository(
  installationId: number,
  owner: string,
  repo: string,
): Promise<GitHubRepository> {
  const { token } = await getInstallationAccessToken(installationId);
  return gitHubRepositorySchema.parse(
    await fetchGitHubApi(`https://api.github.com/repos/${owner}/${repo}`, token),
  );
}

export async function getRepositoryById(
  installationId: number,
  repositoryId: number,
): Promise<GitHubRepository | null> {
  const { token } = await getInstallationAccessToken(installationId);

  const response = await fetch(`https://api.github.com/repositories/${repositoryId}`, {
    headers: {
      Authorization: `Bearer ${token}`,
      ...GITHUB_API_HEADERS,
    },
  });

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to get repository: ${error}`);
  }

  return gitHubRepositorySchema.parse(await response.json());
}
