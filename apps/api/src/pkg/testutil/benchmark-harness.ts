import { z } from "zod";
import { benchmarkTestEnv } from "./env";
import { Harness } from "./harness";
import { StepRequest, StepResponse, step } from "./request";

export class BenchmarkHarness extends Harness {
  public readonly env: z.infer<typeof benchmarkTestEnv>;

  private constructor() {
    super();
    this.env = benchmarkTestEnv.parse(process.env);
  }

  static async init(): Promise<BenchmarkHarness> {
    const h = new BenchmarkHarness();
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
