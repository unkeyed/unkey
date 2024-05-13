import { createExecutionContext, env, waitOnExecutionContext } from "cloudflare:test";
import type { TaskContext } from "vitest";
import "../../worker";
import worker from "../../worker";
import type { Api, KeyAuth, Workspace } from "../db";
import { type Env, zEnv } from "../env";
import { Harness } from "./harness";
import { type StepRequest, type StepResponse, headersToRecord } from "./request";

export type Resources = {
  unkeyWorkspace: Workspace;
  unkeyApi: Api;
  unkeyKeyAuth: KeyAuth;
  userWorkspace: Workspace;
  userApi: Api;
  userKeyAuth: KeyAuth;
};

export class RouteHarness extends Harness {
  private constructor(t: TaskContext) {
    super(t);
  }

  static async init(t: TaskContext): Promise<RouteHarness> {
    const h = new RouteHarness(t);
    await h.seed();
    return h;
  }

  public async do<TRequestBody = unknown, TResponseBody = unknown>(
    req: StepRequest<TRequestBody>,
  ): Promise<StepResponse<TResponseBody>> {
    const ctx = createExecutionContext();

    // we need to add localhost, otherwise hono will not match the routes
    const request = new Request(`http://localhost${req.url}`, {
      method: req.method,
      headers: req.headers,
      body: JSON.stringify(req.body),
    });

    const res = await worker.fetch(
      request,
      {
        DATABASE_HOST: "localhost:3900",
        DATABASE_USERNAME: "unkey",
        DATABASE_PASSWORD: "password",
        DATABASE_NAME: "unkey",
        VERSION: "dev",
        ENVIRONMENT: "production",
        ...env,
      } as Env,
      ctx,
    );

    // const res = await worker.fetch!(request, env as any, ctx);

    await waitOnExecutionContext(ctx);

    const text = await res.text();
    try {
      return {
        status: res.status,
        headers: headersToRecord(res.headers),
        body: JSON.parse(text),
      };
    } catch (err) {
      console.error(`${req.url} didn't return json`, text, err);
      return {
        status: res.status,
        headers: headersToRecord(res.headers),
        body: {} as TResponseBody,
      };
    }
  }

  async get<TRes>(req: Omit<StepRequest<never>, "method">): Promise<StepResponse<TRes>> {
    return await this.do<never, TRes>({ method: "GET", ...req });
  }
  async post<TReq, TRes>(req: Omit<StepRequest<TReq>, "method">): Promise<StepResponse<TRes>> {
    return await this.do<TReq, TRes>({ method: "POST", ...req });
  }
  async put<TReq, TRes>(req: Omit<StepRequest<TReq>, "method">): Promise<StepResponse<TRes>> {
    return await this.do<TReq, TRes>({ method: "PUT", ...req });
  }
  async delete<TRes>(req: Omit<StepRequest<never>, "method">): Promise<StepResponse<TRes>> {
    return await this.do<never, TRes>({ method: "DELETE", ...req });
  }
}
