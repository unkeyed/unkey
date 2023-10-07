import { Result, result } from "@unkey/result";
import { z } from "zod";

type VercelErrorResponse = {
  error: string;
  message: string;
};

const projectSchema = z.object({
  id: z.string(),
  name: z.string(),
});
export type Project = z.infer<typeof projectSchema>;

const environmentVariable = z.object({});
export type EnvironmentVariable = z.infer<typeof environmentVariable>;

export class Vercel {
  private readonly baseUrl: string;
  private readonly token: string;
  private readonly teamId: string | null;

  constructor(opts: { accessToken: string; baseUrl?: string; teamId?: string }) {
    this.baseUrl = opts.baseUrl ?? "https://api.vercel.com";
    this.token = opts.accessToken;
    this.teamId = opts.teamId ?? null;
  }

  private async fetch<TResult>(req: {
    method: "GET" | "POST" | "PUT" | "DELETE";
    path: string[];
    parameters?: Record<string, unknown>;
    opts?: { cache?: RequestCache; revalidate?: number };
    body?: unknown;
  }): Promise<Result<TResult, VercelErrorResponse>> {
    try {
      const url = new URL(req.path.join("/"), this.baseUrl);
      if (req.parameters) {
        for (const [key, value] of Object.entries(req.parameters)) {
          if (typeof value === "undefined" || value === null) {
            continue;
          }
          url.searchParams.set(key, value.toString());
        }
      }
      if (this.teamId) {
        url.searchParams.set("teamId", this.teamId);
      }
      const res = await fetch(url, {
        method: req.method,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${this.token}`,
        },
        body: req.body ? JSON.stringify(req.body) : undefined,
        cache: req.opts?.cache,
        // @ts-ignore
        next: {
          revalidate: req.opts?.revalidate,
        },
      });
      if (!res.ok) {
        const error = (await res.json()) as VercelErrorResponse;
        console.error(error);
        return result.fail(error);
      }
      const body = await res.json();
      // @ts-ignore
      return result.success(body);
    } catch (e) {
      return result.fail({ error: "unexpected catch", message: (e as Error).message });
    }
  }

  public async getProject(projectId: string): Promise<Result<Project, VercelErrorResponse>> {
    return this.fetch({
      method: "GET",
      path: ["v9", "projects", projectId],
    });
  }

  public async listProjects(): Promise<Result<Project[], VercelErrorResponse>> {
    const res = await this.fetch<{ projects: Project[] }>({
      method: "GET",
      path: ["v9", "projects"],
    });
    if (res.error) {
      return result.fail(res.error);
    }
    // @ts-ignore
    return result.success(res.value.projects);
  }

  public async upsertEnvironmentVariable(
    projectId: string,
    environment: string,
    key: string,
    value: string,
    sensitive?: boolean,
  ): Promise<Result<{ created: { id: string } }, VercelErrorResponse>> {
    return await this.fetch({
      method: "POST",
      path: ["v10", "projects", projectId, "env"],
      parameters: { upsert: true },
      body: {
        key,
        value,
        type: sensitive ? (environment === "development" ? "encrypted" : "sensitive") : "plain",
        target: [environment],
      },
    });
  }

  public async removeEnvironmentVariable(
    projectId: string,
    envId: string,
  ): Promise<Result<void, VercelErrorResponse>> {
    return await this.fetch({
      method: "DELETE",
      path: ["v10", "projects", projectId, "env", envId],
    });
  }
}
