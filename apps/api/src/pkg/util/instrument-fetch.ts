import type { Context } from "../hono/app";
import { LogdrainMetrics } from "../metrics/logdrain";

export function instrumentedFetch(c?: Context) {
  return async (
    input: RequestInfo | URL,
    init?: RequestInit<RequestInitCfProperties>,
  ): Promise<Response> => {
    const metrics = c
      ? c.get("services").metrics
      : new LogdrainMetrics({
          isolateId: "unknown",
          requestId: "unknown",
          environment: "unknown",
        });

    const start = performance.now();
    let status = 0;
    try {
      const res = await fetch(input, init);
      status = res.status;
      return res;
    } finally {
      const latency = performance.now() - start;

      metrics.emit({
        metric: "metric.fetch.egress",
        url: input.toString(),
        latency,
        status,
      });
    }
  };
}
