import { type Api, type KeyAuth, type Workspace } from "../db";
import { App, newApp } from "../hono/app";
import { Harness } from "./harness";
import { StepRequest, StepResponse, fetchRoute } from "./request";

export type Resources = {
  unkeyWorkspace: Workspace;
  unkeyApi: Api;
  unkeyKeyAuth: KeyAuth;
  userWorkspace: Workspace;
  userApi: Api;
  userKeyAuth: KeyAuth;
};

export class RouteHarness extends Harness implements Disposable {
  public readonly app: App;

  constructor() {
    super();
    this.app = newApp();
  }

  public useRoutes(...registerFunctions: ((app: App) => any)[]): void {
    registerFunctions.forEach((fn) => fn(this.app));
  }

  async do<TReq, TRes>(req: StepRequest<TReq>): Promise<StepResponse<TRes>> {
    return await fetchRoute<TReq, TRes>(this.app, req);
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
