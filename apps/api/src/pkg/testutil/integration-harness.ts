import type { TaskContext } from "vitest";
import { integrationTestEnv } from "./env";
import { Harness } from "./harness";
import { type StepRequest, type StepResponse, step } from "./request";

export class IntegrationHarness extends Harness {
  public readonly baseUrl: string;

  private constructor(t: TaskContext) {
    super(t);
    this.baseUrl = integrationTestEnv.parse(process.env).UNKEY_BASE_URL;
  }

  static async init(t: TaskContext): Promise<IntegrationHarness> {
    const h = new IntegrationHarness(t);
    await h.seed();
    return h;
  }
  async get<TRes>(req: Omit<StepRequest<never>, "method">): Promise<StepResponse<TRes>> {
    return await step<never, TRes>({ method: "GET", ...req });
  }
  async post<TReq, TRes>(req: Omit<StepRequest<TReq>, "method">): Promise<StepResponse<TRes>> {
    return await step<TReq, TRes>({ method: "POST", ...req });
  }
  async put<TReq, TRes>(req: Omit<StepRequest<TReq>, "method">): Promise<StepResponse<TRes>> {
    return await step<TReq, TRes>({ method: "PUT", ...req });
  }
  async delete<TRes>(req: Omit<StepRequest<never>, "method">): Promise<StepResponse<TRes>> {
    return await step<never, TRes>({ method: "DELETE", ...req });
  }
}
