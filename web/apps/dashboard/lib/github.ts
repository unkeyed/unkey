import crypto from "node:crypto";
import { githubAppEnv } from "@/lib/env";

export type GitHubRepository = {
  id: number;
  name: string;
  full_name: string;
  private: boolean;
  html_url: string;
  default_branch: string;
};

type InstallationAccessToken = {
  token: string;
  expires_at: string;
};

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
  const signature = sign.sign(env.GITHUB_APP_PRIVATE_KEY, "base64url");

  return `${signatureInput}.${signature}`;
}

export async function getInstallationAccessToken(
  installationId: string,
): Promise<InstallationAccessToken> {
  const jwt = generateAppJWT();

  const response = await fetch(
    `https://api.github.com/app/installations/${installationId}/access_tokens`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${jwt}`,
        Accept: "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
      },
    },
  );

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to get installation access token: ${error}`);
  }

  return response.json();
}

export async function getInstallationRepositories(
  installationId: string,
): Promise<GitHubRepository[]> {
  const { token } = await getInstallationAccessToken(installationId);

  const allRepositories: GitHubRepository[] = [];
  let page = 1;
  const perPage = 100;

  while (true) {
    const response = await fetch(
      `https://api.github.com/installation/repositories?per_page=${perPage}&page=${page}`,
      {
        headers: {
          Authorization: `Bearer ${token}`,
          Accept: "application/vnd.github+json",
          "X-GitHub-Api-Version": "2022-11-28",
        },
      },
    );

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Failed to get repositories: ${error}`);
    }

    const data = await response.json();
    allRepositories.push(...data.repositories);

    if (data.repositories.length < perPage) {
      break;
    }
    page++;
  }

  return allRepositories;
}
