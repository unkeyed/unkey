import { integrationTestEnv } from "./env";
import { Harness } from "./harness";
import { StepRequest, StepResponse, step } from "./request";

export class IntegrationHarness extends Harness {
  public readonly baseUrl: string;

  constructor() {
    super();

    this.baseUrl = integrationTestEnv.parse(process.env).UNKEY_BASE_URL;
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
