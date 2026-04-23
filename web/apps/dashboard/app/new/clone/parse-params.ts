import { envVarKeySchema } from "@/lib/schemas/env-var";
import { slugify } from "@/lib/slugify";
import { z } from "zod";

export { slugify };

const githubUrl = /^https?:\/\/github\.com\/([^/\s]+)\/([^/\s#?]+?)(?:\.git)?\/?$/;
const ownerRepo = /^([^/\s]+)\/([^/\s]+?)(?:\.git)?$/;

export type ParsedRepository = {
  owner: string;
  repo: string;
  fullName: string;
  url: string;
};

function parseRepository(raw: string): ParsedRepository | null {
  const trimmed = raw.trim();
  const url = trimmed.match(githubUrl);
  if (url) {
    const [, owner, repo] = url;
    return {
      owner,
      repo,
      fullName: `${owner}/${repo}`,
      url: `https://github.com/${owner}/${repo}`,
    };
  }
  const short = trimmed.match(ownerRepo);
  if (short) {
    const [, owner, repo] = short;
    return {
      owner,
      repo,
      fullName: `${owner}/${repo}`,
      url: `https://github.com/${owner}/${repo}`,
    };
  }
  return null;
}

const rawSchema = z.object({
  repository: z.string().min(1),
  "project-name": z.string().trim().min(1).max(256).optional(),
  branch: z.string().trim().min(1).max(256).optional(),
  "root-directory": z.string().trim().min(1).max(500).optional(),
  dockerfile: z.string().trim().min(1).max(500).optional(),
  env: z.string().optional(),
  envDescription: z.string().trim().max(1024).optional(),
  envLink: z.string().trim().url().optional(),
});

export type CloneParams = {
  repository: ParsedRepository;
  projectName: string | null;
  branch: string | null;
  rootDirectory: string | null;
  dockerfile: string | null;
  envKeys: string[];
  envDescription: string | null;
  envLink: string | null;
};

export type ParseResult = { ok: true; params: CloneParams } | { ok: false; error: string };

function firstString(value: string | string[] | undefined): string | undefined {
  if (Array.isArray(value)) {
    return value[0];
  }
  return value;
}

export function parseCloneParams(
  search: Record<string, string | string[] | undefined>,
): ParseResult {
  const normalized: Record<string, string | undefined> = {};
  for (const key of Object.keys(rawSchema.shape)) {
    normalized[key] = firstString(search[key]);
  }

  const parsed = rawSchema.safeParse(normalized);
  if (!parsed.success) {
    const issue = parsed.error.issues[0];
    const field = issue.path.join(".") || "query";
    return { ok: false, error: `Invalid ${field}: ${issue.message}` };
  }

  const repository = parseRepository(parsed.data.repository);
  if (!repository) {
    return {
      ok: false,
      error: "repository must be a GitHub URL (https://github.com/owner/repo) or owner/repo",
    };
  }

  const envKeys: string[] = [];
  if (parsed.data.env) {
    const seen = new Set<string>();
    for (const raw of parsed.data.env.split(",")) {
      const key = raw.trim();
      if (!key) {
        continue;
      }
      const keyCheck = envVarKeySchema.safeParse(key);
      if (!keyCheck.success) {
        return {
          ok: false,
          error: `Invalid env var key "${key}": ${keyCheck.error.issues[0].message}`,
        };
      }
      if (!seen.has(key)) {
        seen.add(key);
        envKeys.push(key);
      }
    }
  }

  return {
    ok: true,
    params: {
      repository,
      projectName: parsed.data["project-name"] ?? null,
      branch: parsed.data.branch ?? null,
      rootDirectory: parsed.data["root-directory"] ?? null,
      dockerfile: parsed.data.dockerfile ?? null,
      envKeys,
      envDescription: parsed.data.envDescription ?? null,
      envLink: parsed.data.envLink ?? null,
    },
  };
}

export function buildCloneSearchString(params: CloneParams): string {
  const sp = new URLSearchParams();
  sp.set("repository", params.repository.url);
  if (params.projectName) {
    sp.set("project-name", params.projectName);
  }
  if (params.branch) {
    sp.set("branch", params.branch);
  }
  if (params.rootDirectory) {
    sp.set("root-directory", params.rootDirectory);
  }
  if (params.dockerfile) {
    sp.set("dockerfile", params.dockerfile);
  }
  if (params.envKeys.length > 0) {
    sp.set("env", params.envKeys.join(","));
  }
  if (params.envDescription) {
    sp.set("envDescription", params.envDescription);
  }
  if (params.envLink) {
    sp.set("envLink", params.envLink);
  }
  return sp.toString();
}
