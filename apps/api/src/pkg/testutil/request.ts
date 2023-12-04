import { type App } from "@/pkg/hono/app";
export type StepRequest<TRequestBody> = {
  url: string;
  method: "POST" | "GET" | "PUT" | "DELETE";
  headers?: Record<string, string>;
  body?: TRequestBody;
};
export type StepResponse<TBody = unknown> = {
  status: number;
  headers: Record<string, string>;
  body: TBody;
};

export async function step<TRequestBody = unknown, TResponseBody = unknown>(
  req: StepRequest<TRequestBody>,
): Promise<StepResponse<TResponseBody>> {
  const res = await fetch(req.url, {
    method: req.method,
    headers: req.headers,
    body: JSON.stringify(req.body),
  });

  return {
    status: res.status,
    headers: Object.fromEntries(res.headers.entries()),
    body: (await res.json().catch((err) => {
      console.error(`${req.url} didn't return json`, err);
      return {};
    })) as TResponseBody,
  };
}

export async function fetchRoute<TRequestBody = unknown, TResponseBody = unknown>(
  app: App,
  req: StepRequest<TRequestBody>,
): Promise<StepResponse<TResponseBody>> {
  /**
   * Hono requires an ExecutionContext to be passed to the app.request function.
   * Otherwise it will throw an error when trying to access the context.
   */
  const noopExecutionContext: ExecutionContext = {
    waitUntil: (_promise: Promise<any>) => {},
    passThroughOnException: () => {},
  };

  const res = await app.request(
    req.url,
    {
      method: req.method,
      headers: req.headers,
      body: JSON.stringify(req.body),
    },
    {},
    noopExecutionContext,
  );

  return {
    status: res.status,
    headers: Object.fromEntries(res.headers.entries()),
    body: (await res.json().catch((err) => {
      console.error(`${req.url} didn't return json`, err);
      return {};
    })) as TResponseBody,
  };
}
